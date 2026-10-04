// verify_code_controller.go：发送验证码接口。
package controller

import (
	"github.com/gin-gonic/gin"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/service"
)

// VerifyCodeController 验证码 controller。
type VerifyCodeController struct {
	svc *service.VerifyCodeService
}

const verifyCodeAPIPrefix = "/verify-code"

// NewVerifyCodeController 构造。
func NewVerifyCodeController(svc *service.VerifyCodeService) *VerifyCodeController {
	return &VerifyCodeController{svc: svc}
}

// RegisterTo 注册路由。
func (v *VerifyCodeController) RegisterTo(g *gin.RouterGroup) {
	rg := g.Group(verifyCodeAPIPrefix)
	rg.POST("", v.Send)
}

// verifyCodeSendBody 发送验证码请求体。
type verifyCodeSendBody struct {
	Channel string `json:"channel" binding:"required"`
	Target  string `json:"target" binding:"required"`
	Scene   string `json:"scene" binding:"required"`
}

// Send 发送验证码。
func (v *VerifyCodeController) Send(c *gin.Context) {
	var body verifyCodeSendBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, err)
		return
	}
	resp, err := v.svc.SendCode(c.Request.Context(), service.SendVerifyCodeReq{
		Channel:  body.Channel,
		Target:   body.Target,
		Scene:    body.Scene,
		ClientIP: clientIP(c),
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, resp)
}
