// register_controller.go：个人注册 + 企业注册接口。
package controller

import (
	"github.com/gin-gonic/gin"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/errs"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/service"
)

// RegisterController 注册 controller。
type RegisterController struct {
	svc *service.RegisterService
}

const registerAPIPrefix = "/register"

// NewRegisterController 构造。
func NewRegisterController(svc *service.RegisterService) *RegisterController {
	return &RegisterController{svc: svc}
}

// RegisterTo 注册路由。
func (r *RegisterController) RegisterTo(g *gin.RouterGroup) {
	rg := g.Group(registerAPIPrefix)
	rg.POST("/personal", r.Personal)
	rg.POST("/organization", r.Organization)
}

// registerPersonalBody 个人注册请求体。
type registerPersonalBody struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Code     string `json:"code" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
}

// Personal 个人注册。
func (r *RegisterController) Personal(c *gin.Context) {
	var body registerPersonalBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, err)
		return
	}
	resp, err := r.svc.RegisterPersonal(c.Request.Context(), service.RegisterPersonalReq{
		Email:    body.Email,
		Phone:    body.Phone,
		Code:     body.Code,
		Password: body.Password,
		Nickname: body.Nickname,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, resp)
}

// registerOrgBody 企业注册请求体。
type registerOrgBody struct {
	Name            string `json:"name" binding:"required"`
	LegalName       string `json:"legal_name"`
	UnifiedCreditNo string `json:"unified_credit_no"`
	Industry        string `json:"industry"`
	Scale           string `json:"scale"`
	Province        string `json:"province"`
	City            string `json:"city"`
	Address         string `json:"address"`
	ContactEmail    string `json:"contact_email"`
	ContactPhone    string `json:"contact_phone"`
}

// Organization 企业注册（登录态）。
func (r *RegisterController) Organization(c *gin.Context) {
	uid := currentUserID(c)
	if uid == 0 {
		respondError(c, errs.ErrUnauth)
		return
	}
	var body registerOrgBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, err)
		return
	}
	resp, err := r.svc.RegisterOrganization(c.Request.Context(), service.RegisterOrganizationReq{
		CurrentUserID:   uid,
		Name:            body.Name,
		LegalName:       body.LegalName,
		UnifiedCreditNo: body.UnifiedCreditNo,
		Industry:        body.Industry,
		Scale:           body.Scale,
		Province:        body.Province,
		City:            body.City,
		Address:         body.Address,
		ContactEmail:    body.ContactEmail,
		ContactPhone:    body.ContactPhone,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, resp)
}
