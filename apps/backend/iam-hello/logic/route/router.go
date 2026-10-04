// Package route 负责把业务路由与通用中间件挂到 gin.Engine 上。
//
// 设计：
//   - Module interface：业务模块的统一注册契约（RegisterTo(group)）。
//     由各 controller struct 实现；route 包对具体业务零感知。
//   - Register(engine, modules...)：注册通用中间件 + 健康检查 +
//     依次调用每个 Module.RegisterTo(api)。
//   - RegisterMetrics(engine)：注册独立的 metrics 路由（独立端口）。
//
// 中间件顺序遵循通用最佳实践：
//
//	Logger(skip 健康检查) -> Recovery -> RequestID -> AccessLog -> 业务子路由
//
// 新增子模块时（机械化）：
//  1. service 包新增 XxxService struct + 方法；
//  2. controller 包新增 XxxController struct + 方法 + RegisterTo，
//     并提供 NewDefaultXxxController() 便利构造函数（内部装配默认 service / dao）；
//  3. logic.App 中 new 出 controller（调用 NewDefaultXxxController()），作为 route.Register 的可变参数传入。
//     route 包本身不需要任何改动。
package route

import (
	"net/http"

	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic/common/middleware"
	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic/config"

	// 空导入：触发 swag 生成的 docs 包 init()，把 spec 注册进 gin-swagger。
	// 文件由 `swag init -g main.go -o ./docs` 生成；首次接入时本目录中
	// 默认提供一个最小占位包，避免在未运行 swag 时编译失败。
	_ "github.com/zhifenghao123/how-dev-iam/iam-hello/docs"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Module 是业务路由模块的统一契约。
//
// 由 controller struct 实现（典型实现：func (c *XxxController) RegisterTo(g *gin.RouterGroup)），
// route.Register 会把所有传入的 Module 挂到 /api 分组下。
type Module interface {
	RegisterTo(group *gin.RouterGroup)
}

// Register 注册通用中间件、健康检查与所有业务模块。
//
// modules 是可变参数：调用方在 logic.App 中按需 new 出各 controller 后传入即可，
// route 包对具体业务零感知。
func Register(engine *gin.Engine, modules ...Module) error {
	// 1) 通用中间件（顺序敏感）
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/healthCheck/status"},
	}))
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestID())
	// 全量请求/响应访问日志：单行 JSON 写入独立文件
	// （由 logging.Setup 创建并通过 logging.AccessWriter() 暴露，
	// 默认 log/<service>-req.log，与主日志分离）。
	engine.Use(middleware.AccessLog(middleware.AccessLogConfig{}))

	// 2) 健康探测（K8s liveness / readiness）
	engine.GET("/healthCheck/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": config.Global().ServiceName,
		})
	})

	// 3) 业务路由分组：把每个 Module 都挂到 /api 下
	api := engine.Group("/api")
	for _, m := range modules {
		m.RegisterTo(api)
	}

	return nil
}

// RegisterMetrics 注册独立的 metrics 引擎路由。
//
// 在独立端口 + 独立 gin.Engine 上挂载，避免业务路由的中间件
// （例如鉴权）影响 Prometheus 抓取。
func RegisterMetrics(engine *gin.Engine) error {
	//engine.Use(gin.Recovery())
	//// 演示用：返回固定 OK；接入 prometheus 时可替换成 gin.WrapH(promhttp.Handler())
	//engine.GET("/metrics", func(c *gin.Context) {
	//	c.String(http.StatusOK, "# iam-hello metrics placeholder\n")
	//})
	//return nil

	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/metrics"},
	}))
	engine.Use(gin.Recovery())
	// 指标接口
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return nil
}

// RegisterSwagger 把 Swagger UI 挂到 /swagger/*any。
//
// 设计：与 health / metrics 同属基础设施路由，不归属任何业务 controller，
// 因此放在 route 包；是否启用由 App 根据 config.EnableSwagger 决定，
// route 包本身不读配置，保持职责单一。
//
// 访问路径：http://<host>:<http_port>/swagger/index.html
//
// 文档来源：由 `swag init -g main.go -o ./docs` 生成的 docs 包，
// 通过本文件顶部的空导入触发其 init() 注册到全局。
func RegisterSwagger(engine *gin.Engine) error {
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return nil
}
