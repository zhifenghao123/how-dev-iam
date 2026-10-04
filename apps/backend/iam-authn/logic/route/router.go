// Package route 负责把业务路由与通用中间件挂到 gin.Engine 上。
//
// 设计：
//   - Module interface：业务模块的统一注册契约（RegisterTo(group)）。
//     由各 controller struct 实现；route 包对具体业务零感知。
//   - Register(engine, modules...)：注册通用中间件 + 健康检查 +
//     依次调用每个 Module.RegisterTo(api)。
//   - RegisterMetrics(engine)：注册独立的 metrics 路由（独立端口）。
//
// 中间件顺序：
//
//	Logger(skip 健康检查) -> Recovery -> RequestID -> AccessLog -> 业务子路由
package route

import (
	"net/http"

	"github.com/zhifenghao123/how-dev-iam/iam-authn/logic/common/middleware"
	"github.com/zhifenghao123/how-dev-iam/iam-authn/logic/config"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Module 是业务路由模块的统一契约。
type Module interface {
	RegisterTo(group *gin.RouterGroup)
}

// Register 注册通用中间件、健康检查与所有业务模块。
func Register(engine *gin.Engine, modules ...Module) error {
	// 1) 通用中间件（顺序敏感）
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/healthCheck/status"},
	}))
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestID())
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
// 影响 Prometheus 抓取。
func RegisterMetrics(engine *gin.Engine) error {
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/metrics"},
	}))
	engine.Use(gin.Recovery())
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return nil
}
