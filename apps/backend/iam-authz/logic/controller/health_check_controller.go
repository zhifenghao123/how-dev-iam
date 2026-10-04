// health_check_controller.go 提供 /api 分组下的健康检查接口。
//
// 与根路径 /healthCheck/status（K8s liveness / readiness）互补：
//   - /healthCheck/status  在 route.Register 中注册，用于探活，
//     被访问日志与 gin logger 跳过；
//   - /api/healthCheck/ping 在此处注册，走完整业务中间件链，
//     便于验证网关到服务的完整链路。
package controller

import (
	"net/http"
	"time"

	"github.com/zhifenghao123/how-dev-iam/iam-authz/logic/config"
	"github.com/zhifenghao123/how-dev-iam/iam-authz/logic/constant"

	"github.com/gin-gonic/gin"
)

// HealthCheckController 是健康检查模块的 controller。
//
// 当前无业务依赖，仅在 App 装配阶段作为最小可用 Module 挂到 /api 分组。
type HealthCheckController struct{}

const healthCheckAPIPrefix = "/healthCheck"

// NewHealthCheckController 构造 HealthCheckController。
func NewHealthCheckController() *HealthCheckController {
	return &HealthCheckController{}
}

// NewDefaultHealthCheckController 默认装配入口，保持与其他 controller 命名一致。
func NewDefaultHealthCheckController() *HealthCheckController {
	return NewHealthCheckController()
}

// RegisterTo 把本模块的路由挂到给定 RouterGroup 下。
//
// 当前路由：
//   - GET /healthCheck/ping  返回服务名 / 时间戳 / 状态
func (hc *HealthCheckController) RegisterTo(group *gin.RouterGroup) {
	g := group.Group(healthCheckAPIPrefix)
	g.GET("/ping", hc.Ping)
}

// Ping 返回一个最小健康信息。
func (hc *HealthCheckController) Ping(c *gin.Context) {
	svcName := constant.ServiceName
	if cfg := config.Global(); cfg != nil && cfg.ServiceName != "" {
		svcName = cfg.ServiceName
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": svcName,
		"time":    time.Now().Format(time.RFC3339Nano),
	})
}
