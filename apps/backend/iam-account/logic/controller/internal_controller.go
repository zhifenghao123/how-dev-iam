// internal_controller.go：供 iam-authn 等内部服务调用的接口。
//   POST /internal/users/lookup           按 email/phone 查 User
//   POST /internal/users/verify-password  校验密码
//
// 内部接口挂在 /internal 前缀下；生产建议由 gateway 层强制内网访问 / mTLS。
package controller

import (
	"github.com/gin-gonic/gin"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/service"
)

// InternalController 内部接口 controller。
type InternalController struct {
	svc *service.InternalService
}

// NewInternalController 构造。
func NewInternalController(svc *service.InternalService) *InternalController {
	return &InternalController{svc: svc}
}

// RegisterTo 注册路由。注意：本 controller 的路由不挂在 /api 下，
// 由 route.Register 特殊注入到 /internal 分组（见 app.go 的装配）。
func (i *InternalController) RegisterTo(g *gin.RouterGroup) {
	rg := g.Group("/users")
	rg.POST("/lookup", i.Lookup)
	rg.POST("/verify-password", i.VerifyPassword)
}

type lookupBody struct {
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	UserID int64  `json:"user_id"`
}

// Lookup 按标识查询用户。
func (i *InternalController) Lookup(c *gin.Context) {
	var body lookupBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, err)
		return
	}
	resp, err := i.svc.LookupUser(c.Request.Context(), service.LookupReq{
		Email: body.Email, Phone: body.Phone, UserID: body.UserID,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, resp)
}

type verifyPasswordBody struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	UserID   int64  `json:"user_id"`
	Password string `json:"password" binding:"required"`
}

// VerifyPassword 校验密码。
func (i *InternalController) VerifyPassword(c *gin.Context) {
	var body verifyPasswordBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, err)
		return
	}
	resp, err := i.svc.VerifyPassword(c.Request.Context(), service.VerifyPasswordReq{
		Email: body.Email, Phone: body.Phone, UserID: body.UserID, Password: body.Password,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, resp)
}
