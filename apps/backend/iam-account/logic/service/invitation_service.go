// invitation_service.go 提供企业邀请成员 / 接受邀请两条能力。
package service

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/errs"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/snowflake"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/config"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/dao"
)

// InvitationService 邀请服务。
type InvitationService struct {
	db            *sql.DB
	cfg           *config.Detail
	accountDao    dao.AccountDao
	membershipDao dao.MembershipDao
	invitationDao dao.InvitationDao
	userDao       dao.UserDao
}

// NewInvitationService 构造。
func NewInvitationService(
	db *sql.DB, cfg *config.Detail,
	a dao.AccountDao, m dao.MembershipDao, i dao.InvitationDao, u dao.UserDao,
) *InvitationService {
	return &InvitationService{db: db, cfg: cfg, accountDao: a, membershipDao: m, invitationDao: i, userDao: u}
}

// CreateInvitationReq 创建邀请入参。
type CreateInvitationReq struct {
	CurrentUserID int64
	AccountID     int64
	Email         string
	Phone         string
	Role          string // admin | member | viewer
}

// CreateInvitation 由企业管理员发起邀请。
func (s *InvitationService) CreateInvitation(ctx context.Context, req CreateInvitationReq) (*InvitationView, error) {
	if req.CurrentUserID == 0 {
		return nil, errs.ErrUnauth
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Email == "" && req.Phone == "" {
		return nil, errs.ErrBadRequest.WithMessage("email or phone is required")
	}
	role := req.Role
	if role == "" {
		role = "member"
	}
	if role != "admin" && role != "member" && role != "viewer" {
		return nil, errs.ErrBadRequest.WithMessage("invalid role")
	}

	// 校验 account
	acc, err := s.accountDao.GetByID(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, errs.ErrAccountNotFound
	}
	if acc.Type != "organization" {
		return nil, errs.ErrAccountNotOrg
	}

	// 校验当前 user 权限（必须是 owner 或 admin）
	m, err := s.membershipDao.GetByAccountAndUser(ctx, req.AccountID, req.CurrentUserID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.Status != "active" || (m.Role != "owner" && m.Role != "admin") {
		return nil, errs.ErrForbidden
	}

	token, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	po := &dao.InvitationPO{
		ID:           snowflake.Next(),
		AccountID:    req.AccountID,
		InviterID:    req.CurrentUserID,
		InviteeEmail: nullString(req.Email),
		InviteePhone: nullString(req.Phone),
		Role:         role,
		Token:        token,
		Status:       "pending",
		ExpireAt:     time.Now().Add(time.Duration(s.cfg.Account.InvitationTTLSeconds) * time.Second),
	}
	if err := s.invitationDao.Insert(ctx, nil, po); err != nil {
		return nil, err
	}
	v := toInvitationView(po)
	return &v, nil
}

// AcceptInvitationReq 接受邀请入参。
type AcceptInvitationReq struct {
	CurrentUserID int64
	Token         string
}

// AcceptInvitation 接受邀请，建立 Membership。
func (s *InvitationService) AcceptInvitation(ctx context.Context, req AcceptInvitationReq) (*MembershipView, error) {
	if req.CurrentUserID == 0 {
		return nil, errs.ErrUnauth
	}
	if req.Token == "" {
		return nil, errs.ErrInvitationInvalid
	}
	inv, err := s.invitationDao.GetByToken(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, errs.ErrInvitationInvalid
	}
	if inv.Status != "pending" {
		return nil, errs.ErrInvitationConsumed
	}
	if time.Now().After(inv.ExpireAt) {
		return nil, errs.ErrInvitationExpired
	}

	// 若已是成员则报冲突
	existed, err := s.membershipDao.GetByAccountAndUser(ctx, inv.AccountID, req.CurrentUserID)
	if err != nil {
		return nil, err
	}
	if existed != nil && existed.Status != "left" {
		return nil, errs.ErrMembershipExists
	}

	// 事务：新增 membership + 更新 invitation
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// 取一下当前用户以便 display_name 兜底
	u, err := s.userDao.GetByID(ctx, req.CurrentUserID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errs.ErrUserNotFound
	}

	membershipID := snowflake.Next()
	mpo := &dao.MembershipPO{
		ID:          membershipID,
		AccountID:   inv.AccountID,
		UserID:      req.CurrentUserID,
		Role:        inv.Role,
		Status:      "active",
		DisplayName: u.Nickname,
		JoinSource:  "invited",
		InvitedBy:   nullInt64(inv.InviterID),
	}
	if err := s.membershipDao.Insert(ctx, tx, mpo); err != nil {
		return nil, err
	}
	if err := s.invitationDao.MarkAccepted(ctx, tx, inv.ID, req.CurrentUserID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	v := toMembershipView(mpo)
	return &v, nil
}
