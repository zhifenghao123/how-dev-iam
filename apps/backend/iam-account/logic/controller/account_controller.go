// account_controller.go：账户相关接口。
//   GET /me/accounts   当前 User 的账户列表
package controller

import (
	"github.com/gin-gonic/gin"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/errs"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/service"
)

// AccountController 账户 controller。
type AccountController struct {
	svc *service.AccountService
}

// NewAccountController 构造。
func NewAccountController(svc *service.AccountService) *AccountController {
	return &AccountController{svc: svc}
}

// RegisterTo 注册路由。
func (a *AccountController) RegisterTo(g *gin.RouterGroup) {
	g.GET("/me/accounts", a.ListMine)
}

// ListMine 我的账户列表。
func (a *AccountController) ListMine(c *gin.Context) {
	uid := currentUserID(c)
	if uid == 0 {
		respondError(c, errs.ErrUnauth)
		return
	}
	resp, err := a.svc.ListMyAccounts(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	respondOK(c, resp)
}
