// internal_service.go 提供给 iam-authn 等内部服务调用的接口：
//   - LookupUser：按 email/phone 查 User
//   - VerifyPassword：校验密码
//
// 这些接口不做前端网关暴露，走内网。
package service

import (
	"context"
	"strings"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/errs"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/password"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/dao"
)

// InternalService 内部接口服务。
type InternalService struct {
	userDao       dao.UserDao
	credentialDao dao.UserCredentialDao
}

// NewInternalService 构造。
func NewInternalService(u dao.UserDao, c dao.UserCredentialDao) *InternalService {
	return &InternalService{userDao: u, credentialDao: c}
}

// LookupReq 查询入参。email / phone / userID 三选一。
type LookupReq struct {
	Email  string
	Phone  string
	UserID int64
}

// LookupUser 按标识查询用户。
func (s *InternalService) LookupUser(ctx context.Context, req LookupReq) (*UserView, error) {
	var po *dao.UserPO
	var err error
	if req.UserID != 0 {
		po, err = s.userDao.GetByID(ctx, req.UserID)
	} else if e := strings.TrimSpace(strings.ToLower(req.Email)); e != "" {
		po, err = s.userDao.GetByEmail(ctx, e)
	} else if p := strings.TrimSpace(req.Phone); p != "" {
		po, err = s.userDao.GetByPhone(ctx, p)
	} else {
		return nil, errs.ErrBadRequest.WithMessage("email, phone or user_id is required")
	}
	if err != nil {
		return nil, err
	}
	if po == nil {
		return nil, errs.ErrUserNotFound
	}
	v := toUserView(po)
	return &v, nil
}

// VerifyPasswordReq 校验密码入参。
type VerifyPasswordReq struct {
	Email    string
	Phone    string
	UserID   int64
	Password string
}

// VerifyPasswordResp 校验密码出参。
type VerifyPasswordResp struct {
	OK   bool     `json:"ok"`
	User UserView `json:"user"`
}

// VerifyPassword 校验密码。
func (s *InternalService) VerifyPassword(ctx context.Context, req VerifyPasswordReq) (*VerifyPasswordResp, error) {
	uv, err := s.LookupUser(ctx, LookupReq{Email: req.Email, Phone: req.Phone, UserID: req.UserID})
	if err != nil {
		return nil, err
	}
	cred, err := s.credentialDao.GetByUserID(ctx, uv.ID)
	if err != nil {
		return nil, err
	}
	if cred == nil {
		return &VerifyPasswordResp{OK: false, User: *uv}, nil
	}
	ok := password.Verify(cred.PasswordHash, req.Password)
	return &VerifyPasswordResp{OK: ok, User: *uv}, nil
}
