// account_service.go 提供"我的账户列表"能力。
package service

import (
	"context"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/errs"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/dao"
)

// AccountService 账户服务。
type AccountService struct {
	accountDao    dao.AccountDao
	membershipDao dao.MembershipDao
}

// NewAccountService 构造。
func NewAccountService(a dao.AccountDao, m dao.MembershipDao) *AccountService {
	return &AccountService{accountDao: a, membershipDao: m}
}

// ListMyAccounts 列出当前 User 参与的所有 Account。
//
// 返回 Membership + Account 展开视图。
func (s *AccountService) ListMyAccounts(ctx context.Context, currentUserID int64) ([]MembershipView, error) {
	if currentUserID == 0 {
		return nil, errs.ErrUnauth
	}
	memberships, err := s.membershipDao.ListByUser(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	out := make([]MembershipView, 0, len(memberships))
	for i := range memberships {
		v := toMembershipView(&memberships[i])
		acc, err := s.accountDao.GetByID(ctx, memberships[i].AccountID)
		if err == nil && acc != nil {
			av := toAccountView(acc)
			v.Account = &av
		}
		out = append(out, v)
	}
	return out, nil
}
