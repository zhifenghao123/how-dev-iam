// invitation_controller.go：邀请相关接口。
//   POST /accounts/{aid}/invitations   企业管理员发起邀请
//   POST /invitations/:token/accept    接受邀请
package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/errs"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/service"
)

// InvitationController 邀请 controller。
type InvitationController struct {
	svc *service.InvitationService
}

// NewInvitationController 构造。
func NewInvitationController(svc *service.InvitationService) *InvitationController {
	return &InvitationController{svc: svc}
}

// RegisterTo 注册路由。
func (i *InvitationController) RegisterTo(g *gin.RouterGroup) {
	g.POST("/accounts/:aid/invitations", i.Create)
	g.POST("/invitations/:token/accept", i.Accept)
}

type createInvitationBody struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Role  string `json:"role"`
}

// Create 发起邀请。
func (i *InvitationController) Create(c *gin.Context) {
	uid := currentUserID(c)
	if uid == 0 {
		respondError(c, errs.ErrUnauth)
		return
	}
	aid, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		respondError(c, errs.ErrBadRequest.WithMessage("invalid account id"))
		return
	}
	var body createInvitationBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, err)
		return
	}
	resp, err := i.svc.CreateInvitation(c.Request.Context(), service.CreateInvitationReq{
		CurrentUserID: uid,
		AccountID:     aid,
		Email:         body.Email,
		Phone:         body.Phone,
		Role:          body.Role,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, resp)
}

// Accept 接受邀请。
func (i *InvitationController) Accept(c *gin.Context) {
	uid := currentUserID(c)
	if uid == 0 {
		respondError(c, errs.ErrUnauth)
		return
	}
	token := c.Param("token")
	resp, err := i.svc.AcceptInvitation(c.Request.Context(), service.AcceptInvitationReq{
		CurrentUserID: uid,
		Token:         token,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, resp)
}
