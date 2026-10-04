// authn_manage_controller.go 是 authn 管理模块的 controller 层：
//   - AuthnManageController struct：持有 service 依赖（构造函数注入）
//   - HTTP handler 作为 struct 方法，仅做入参提取与响应封装，不含业务逻辑
//   - RegisterTo(group)：把本模块路由挂到给定 RouterGroup 下，
//     是 route.Module 接口的实现，便于在 route.Register 中以可变参数统一注入。
//
// 具体业务逻辑在 logic/service/authn_manage_service.go 中。
package controller

import (
	"net/http"

	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic/service"

	"github.com/gin-gonic/gin"
)

// AuthnManageController 是 authn 管理模块的 controller。
//
// 字段：
//   - svc：业务服务，由构造函数注入。
type AuthnManageController struct {
	svc *service.AuthnManageService
}

const authnApiPrefix = "/authn"

// NewAuthnManageController 构造 AuthnManageController。依赖通过参数显式注入。
func NewAuthnManageController(svc *service.AuthnManageService) *AuthnManageController {
	return &AuthnManageController{svc: svc}
}

// NewDefaultAuthnManageController 使用默认 service 装配 AuthnManageController。
//
// 遵循脚手架的"默认装配"约定：把整条默认装配链（dao -> service -> controller）
// 封装在模块内部，让 logic.App 只需与 controller 对话，
// 不必感知 service / dao 的具体类型。
func NewDefaultAuthnManageController() *AuthnManageController {
	return NewAuthnManageController(service.NewDefaultAuthnManageService())
}

// RegisterTo 把本模块的路由挂到给定 RouterGroup 下。
//
// 当前路由：
//   - GET /authn/supportTypes  返回支持的认证方式列表
func (ac *AuthnManageController) RegisterTo(group *gin.RouterGroup) {
	g := group.Group(authnApiPrefix)
	g.GET("/supportTypes", ac.SupportType)
}

// SupportType 返回当前服务支持的认证方式列表。
//
// @Summary  支持的认证方式列表
// @Description  返回当前服务内置支持的认证方式（password / sms / oauth2 等）。
// @Tags     authn
// @Produce  json
// @Success  200  {object}  map[string]interface{}  "包含 types 数组，每项含 code / name / description"
// @Failure  500  {object}  map[string]string       "内部错误"
// @Router   /authn/supportTypes [get]
func (ac *AuthnManageController) SupportType(c *gin.Context) {
	types, err := ac.svc.SupportTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"types": types})
}
