// Package route 负责把业务路由与通用中间件挂到 gin.Engine 上。
//
// 设计：
//   - Module interface：业务模块的统一注册契约（RegisterTo(group)）。
//   - Register(engine, publicModules, internalModules)：
//     - CORS + Logger(skip 健康检查) + Recovery + RequestID + AccessLog
//     - /healthCheck/status 探活
//     - /api 分组：挂 publicModules
//     - /internal 分组：挂 internalModules（无 CORS 差异；生产由网关限内网）
//   - RegisterMetrics(engine)：独立 metrics 端口路由。
package route

import (
	"net/http"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/middleware"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/config"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Module 是业务路由模块的统一契约。
type Module interface {
	RegisterTo(group *gin.RouterGroup)
}

// Register 注册通用中间件、健康检查与所有业务模块。
//
// publicModules 挂到 /api，internalModules 挂到 /internal。
func Register(engine *gin.Engine, publicModules []Module, internalModules []Module) error {
	// 1) 通用中间件（顺序敏感）
	// CORS 尽可能靠外，让 OPTIONS 预检请求不走到后续业务逻辑。
	engine.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: false,
	}))
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

	// 3) 业务路由分组：/api
	api := engine.Group("/api")
	for _, m := range publicModules {
		m.RegisterTo(api)
	}

	// 4) 内部路由：/internal（生产由网关强制内网 / mTLS）
	internal := engine.Group("/internal")
	for _, m := range internalModules {
		m.RegisterTo(internal)
	}

	return nil
}

// RegisterMetrics 注册独立的 metrics 引擎路由。
func RegisterMetrics(engine *gin.Engine) error {
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/metrics"},
	}))
	engine.Use(gin.Recovery())
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return nil
}
