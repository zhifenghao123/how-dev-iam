// Package logic 是脚手架的"聚合根"。
//
// App 承担三件事：
//  1. 持有共享资源（配置 / HTTP server / metrics server / 计数器）。
//  2. 通过 controllers() 方法集中管理要注入路由的业务 controller 集合，
//     并把 controller 作为 route.Module 注入到 HTTP 引擎中。
//  3. 提供生命周期方法：New / Listen* / Shutdown / WaitForActiveRequests / Close。
//
// App 不持有任何 controller 字段，只在 controllers() 方法内即用即建；
// route 包对具体业务零感知。
package logic

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/zhifenghao123/how-dev-iam/iam-authv/logic/config"
	"github.com/zhifenghao123/how-dev-iam/iam-authv/logic/constant"
	"github.com/zhifenghao123/how-dev-iam/iam-authv/logic/controller"
	"github.com/zhifenghao123/how-dev-iam/iam-authv/logic/route"

	"github.com/gin-gonic/gin"
)

// App 是脚手架的聚合根。
type App struct {
	cfg *config.Detail

	HTTPEngine    *gin.Engine
	MetricsEngine *gin.Engine

	httpServer    *http.Server
	metricsServer *http.Server

	// 活跃请求计数（用于优雅关闭时等长流式连接完成）
	activeReq atomic.Int64
}

// New 装配 App 实例。
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
	// 包一层中间件：在最外层维护 activeReq 计数，便于优雅关闭等待。
	g.Use(a.activeRequestCounter())
	if err := route.Register(g, a.controllers()...); err != nil {
		return err
	}
	// Swagger UI 当前脚手架尚未接入 swag 生成，仅提示。
	if a.cfg.EnableSwagger {
		log.Printf("[%s] enable_swagger=true, but swagger docs not wired yet; skip", constant.ServiceName)
	}
	a.HTTPEngine = g
	return nil
}

// controllers 集中管理要注入到 HTTP 路由的所有业务 controller。
//
// 新增业务模块时，只需在此切片中追加一行 NewDefaultXxxController()。
func (a *App) controllers() []route.Module {
	return []route.Module{
		controller.NewDefaultHealthCheckController(),
		// 新增业务模块时，在此追加一行：
		// controller.NewDefaultXxxController(),
	}
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
	// TODO: 后续接入 dao/db 后在此关闭资源。
	log.Printf("[%s] close ......", constant.ServiceName)
}
