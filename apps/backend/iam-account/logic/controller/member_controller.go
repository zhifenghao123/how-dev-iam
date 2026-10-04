// member_controller.go：企业成员管理接口。
//   POST   /accounts/:aid/members       企业管理员创建子用户
//   GET    /accounts/:aid/members       列成员
//   DELETE /accounts/:aid/members/:mid  移除成员
package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/errs"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/service"
)

// MemberController 成员管理 controller。
type MemberController struct {
	svc *service.MemberService
}

// NewMemberController 构造。
func NewMemberController(svc *service.MemberService) *MemberController {
	return &MemberController{svc: svc}
}

// RegisterTo 注册路由。
func (m *MemberController) RegisterTo(g *gin.RouterGroup) {
	rg := g.Group("/accounts/:aid/members")
	rg.POST("", m.Create)
	rg.GET("", m.List)
	rg.DELETE("/:mid", m.Remove)
}

type createMemberBody struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
	Nickname string `json:"nickname"`
	RealName string `json:"real_name"`
}

// Create 创建子用户。
func (m *MemberController) Create(c *gin.Context) {
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
	var body createMemberBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, err)
		return
	}
	resp, err := m.svc.CreateMember(c.Request.Context(), service.CreateMemberReq{
		CurrentUserID: uid,
		AccountID:     aid,
		Email:         body.Email,
		Phone:         body.Phone,
		Password:      body.Password,
		Role:          body.Role,
		Nickname:      body.Nickname,
		RealName:      body.RealName,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, resp)
}

// List 列成员。
func (m *MemberController) List(c *gin.Context) {
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
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	resp, err := m.svc.ListMembers(c.Request.Context(), service.ListMembersReq{
		CurrentUserID: uid,
		AccountID:     aid,
		Offset:        offset,
		Limit:         limit,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, resp)
}

// Remove 移除成员。
func (m *MemberController) Remove(c *gin.Context) {
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
	mid, err := strconv.ParseInt(c.Param("mid"), 10, 64)
	if err != nil {
		respondError(c, errs.ErrBadRequest.WithMessage("invalid membership id"))
		return
	}
	if err := m.svc.RemoveMember(c.Request.Context(), service.RemoveMemberReq{
		CurrentUserID: uid,
		AccountID:     aid,
		MembershipID:  mid,
	}); err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, gin.H{})
}
