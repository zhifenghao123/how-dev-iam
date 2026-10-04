// authz_manage_controller.go 是 authz 管理模块的 controller 层：
//   - AuthzManageController struct：持有 service 依赖（构造函数注入）
//   - HTTP handler 作为 struct 方法，仅做入参提取与响应封装，不含业务逻辑
//   - RegisterTo(group)：把本模块路由挂到给定 RouterGroup 下，
//     是 route.Module 接口的实现，便于在 route.Register 中以可变参数统一注入。
//
// 具体业务逻辑在 logic/service/authz_manage_service.go 中。
package controller

import (
	"net/http"

	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic/service"

	"github.com/gin-gonic/gin"
)

// AuthzManageController 是 authz 管理模块的 controller。
type AuthzManageController struct {
	svc *service.AuthzManageService
}

const authzApiPrefix = "/authz"

// NewAuthzManageController 构造 AuthzManageController。依赖通过参数显式注入。
func NewAuthzManageController(svc *service.AuthzManageService) *AuthzManageController {
	return &AuthzManageController{svc: svc}
}

// NewDefaultAuthzManageController 使用默认 service 装配 AuthzManageController。
func NewDefaultAuthzManageController() *AuthzManageController {
	return NewAuthzManageController(service.NewDefaultAuthzManageService())
}

// RegisterTo 把本模块的路由挂到给定 RouterGroup 下。
//
// 当前路由：
//   - GET /authz/supportTypes  返回支持的授权方式列表
func (ac *AuthzManageController) RegisterTo(group *gin.RouterGroup) {
	g := group.Group(authzApiPrefix)
	g.GET("/supportTypes", ac.SupportType)
}

// SupportType 返回当前服务支持的授权方式列表。
//
// @Summary  支持的授权方式列表
// @Description  返回当前服务内置支持的授权方式（rbac / abac / acl / rebac 等）。
// @Tags     authz
// @Produce  json
// @Success  200  {object}  map[string]interface{}  "包含 types 数组，每项含 code / name / description"
// @Failure  500  {object}  map[string]string       "内部错误"
// @Router   /authz/supportTypes [get]
func (ac *AuthzManageController) SupportType(c *gin.Context) {
	types, err := ac.svc.SupportTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"types": types})
}
