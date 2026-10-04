// Package logic 是脚手架的"聚合根"。
//
// App 承担三件事：
//  1. 持有共享资源（配置 / MySQL 连接池 / HTTP server / metrics server / 计数器）。
//  2. 装配业务 controller（public + internal），注入 HTTP 路由。
//  3. 提供生命周期方法：New / Listen* / Shutdown / WaitForActiveRequests / Close。
package logic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/config"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/constant"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/controller"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/dao"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/route"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/service"

	"github.com/gin-gonic/gin"
)

// App 是脚手架的聚合根。
type App struct {
	cfg *config.Detail

	db *sql.DB

	HTTPEngine    *gin.Engine
	MetricsEngine *gin.Engine

	httpServer    *http.Server
	metricsServer *http.Server

	// 活跃请求计数（用于优雅关闭时等长流式连接完成）
	activeReq atomic.Int64
}

// New 装配 App 实例（从文件加载配置）。
func New(configPath string) (*App, error) {
	cfg, err := config.New(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return NewFromConfig(cfg)
}

// NewFromConfig 基于已加载的配置装配 App。
func NewFromConfig(cfg *config.Detail) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("app.NewFromConfig: cfg is nil")
	}

	a := &App{cfg: cfg}

	// 1) 打开 MySQL 连接池（建表由 K8s init-job 完成，服务只 Ping）。
	db, err := dao.OpenMySQL(cfg.MySQL)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	a.db = db
	log.Printf("[%s] mysql connected: max_open=%d max_idle=%d",
		constant.ServiceName, cfg.MySQL.MaxOpenConns, cfg.MySQL.MaxIdleConns)

	if err := a.initHTTPEngine(); err != nil {
		return nil, fmt.Errorf("init http engine: %w", err)
	}
	if err := a.initMetricsEngine(); err != nil {
		return nil, fmt.Errorf("init metrics engine: %w", err)
	}
	return a, nil
}

func (a *App) initHTTPEngine() error {
	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	g.Use(a.activeRequestCounter())

	publicModules, internalModules := a.buildControllers()
	if err := route.Register(g, publicModules, internalModules); err != nil {
		return err
	}
	if a.cfg.EnableSwagger {
		log.Printf("[%s] enable_swagger=true, but swagger docs not wired yet; skip", constant.ServiceName)
	}
	a.HTTPEngine = g
	return nil
}

// buildControllers 集中装配 dao -> service -> controller 依赖链。
//
// 返回 (公开 modules, 内部 modules)。
func (a *App) buildControllers() ([]route.Module, []route.Module) {
	// ---------- dao ----------
	userDao := dao.NewUserMySQLDao(a.db)
	credentialDao := dao.NewUserCredentialMySQLDao(a.db)
	accountDao := dao.NewAccountMySQLDao(a.db)
	orgProfileDao := dao.NewAccountOrgProfileMySQLDao(a.db)
	membershipDao := dao.NewMembershipMySQLDao(a.db)
	invitationDao := dao.NewInvitationMySQLDao(a.db)
	verifyCodeDao := dao.NewVerifyCodeMySQLDao(a.db)

	// ---------- service ----------
	verifyCodeSvc := service.NewVerifyCodeService(verifyCodeDao, a.cfg)
	registerSvc := service.NewRegisterService(a.db, a.cfg, verifyCodeSvc,
		userDao, credentialDao, accountDao, orgProfileDao, membershipDao)
	invitationSvc := service.NewInvitationService(a.db, a.cfg,
		accountDao, membershipDao, invitationDao, userDao)
	memberSvc := service.NewMemberService(a.db, a.cfg,
		userDao, credentialDao, accountDao, membershipDao)
	accountSvc := service.NewAccountService(accountDao, membershipDao)
	internalSvc := service.NewInternalService(userDao, credentialDao)

	// ---------- controller ----------
	publicModules := []route.Module{
		controller.NewDefaultHealthCheckController(),
		controller.NewVerifyCodeController(verifyCodeSvc),
		controller.NewRegisterController(registerSvc),
		controller.NewInvitationController(invitationSvc),
		controller.NewMemberController(memberSvc),
		controller.NewAccountController(accountSvc),
	}
	internalModules := []route.Module{
		controller.NewInternalController(internalSvc),
	}
	return publicModules, internalModules
}

func (a *App) initMetricsEngine() error {
	g := gin.New()
	if err := route.RegisterMetrics(g); err != nil {
		return err
	}
	a.MetricsEngine = g
	return nil
}

// activeRequestCounter 维护活跃请求计数。
func (a *App) activeRequestCounter() gin.HandlerFunc {
	return func(c *gin.Context) {
		a.activeReq.Add(1)
		defer a.activeReq.Add(-1)
		c.Next()
	}
}

// ===== 启动 =====

// ListenHTTP 在配置端口上启动 HTTP server，阻塞直到出错或被 Shutdown。
func (a *App) ListenHTTP(ctx context.Context) error {
	a.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.cfg.HTTPPort),
		Handler: a.HTTPEngine,
	}
	log.Printf("[%s] HTTP server starting on :%d", constant.ServiceName, a.cfg.HTTPPort)
	if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// ListenMetrics 在独立端口上启动 metrics server。
func (a *App) ListenMetrics(ctx context.Context) error {
	a.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.cfg.MetricsPort),
		Handler: a.MetricsEngine,
	}
	log.Printf("[%s] Metrics server starting on :%d", constant.ServiceName, a.cfg.MetricsPort)
	if err := a.metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// ===== 优雅关闭 =====

// Shutdown 停止接收新请求。
func (a *App) Shutdown(ctx context.Context) {
	if a.httpServer != nil {
		_ = a.httpServer.Shutdown(ctx)
	}
	if a.metricsServer != nil {
		_ = a.metricsServer.Shutdown(ctx)
	}
}

// ActiveRequestCount 返回当前活跃请求数。
func (a *App) ActiveRequestCount() int64 { return a.activeReq.Load() }

// WaitForActiveRequests 等待活跃请求归零，或 ctx 超时。
func (a *App) WaitForActiveRequests(ctx context.Context) bool {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()
	for {
		if a.activeReq.Load() == 0 {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-t.C:
		}
	}
}

// Close 关闭所有持有资源。
func (a *App) Close() {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			log.Printf("[%s] close db err: %v", constant.ServiceName, err)
		}
		a.db = nil
	}
	log.Printf("[%s] close done", constant.ServiceName)
}
