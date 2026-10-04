// member_service.go 提供企业管理员在企业内直接创建子用户 + 列表 + 移除能力。
package service

import (
	"context"
	"database/sql"
	"strings"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/errs"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/password"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/snowflake"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/config"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/dao"
)

// MemberService 企业成员管理服务。
type MemberService struct {
	db            *sql.DB
	cfg           *config.Detail
	userDao       dao.UserDao
	credentialDao dao.UserCredentialDao
	accountDao    dao.AccountDao
	membershipDao dao.MembershipDao
}

// NewMemberService 构造。
func NewMemberService(
	db *sql.DB, cfg *config.Detail,
	u dao.UserDao, c dao.UserCredentialDao, a dao.AccountDao, m dao.MembershipDao,
) *MemberService {
	return &MemberService{db: db, cfg: cfg, userDao: u, credentialDao: c, accountDao: a, membershipDao: m}
}

// CreateMemberReq 由企业管理员创建子用户。
type CreateMemberReq struct {
	CurrentUserID int64
	AccountID     int64
	Email         string
	Phone         string
	Password      string // 初始密码，首次登录强制改
	Role          string // admin | member | viewer
	Nickname      string
	RealName      string
}

// CreateMember 企业管理员创建子用户。
//
// 若 email/phone 已对应存在的 User：直接建 Membership（相当于隐式邀请）。
// 若不存在：新建 User + Credential + Personal Account + 目标企业 Membership。
func (s *MemberService) CreateMember(ctx context.Context, req CreateMemberReq) (*MembershipView, error) {
	if req.CurrentUserID == 0 {
		return nil, errs.ErrUnauth
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Email == "" && req.Phone == "" {
		return nil, errs.ErrBadRequest.WithMessage("email or phone is required")
	}
	if len(req.Password) < 8 {
		return nil, errs.ErrBadRequest.WithMessage("password must be at least 8 chars")
	}
	role := req.Role
	if role == "" {
		role = "member"
	}
	if role != "admin" && role != "member" && role != "viewer" {
		return nil, errs.ErrBadRequest.WithMessage("invalid role")
	}

	// 权限校验
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
	curM, err := s.membershipDao.GetByAccountAndUser(ctx, req.AccountID, req.CurrentUserID)
	if err != nil {
		return nil, err
	}
	if curM == nil || curM.Status != "active" || (curM.Role != "owner" && curM.Role != "admin") {
		return nil, errs.ErrForbidden
	}

	// 成员数上限
	cnt, err := s.membershipDao.CountByAccount(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	if acc.MemberLimit > 0 && cnt >= acc.MemberLimit {
		return nil, errs.ErrMemberLimitExceed
	}

	// 查已存在 User
	var existedUser *dao.UserPO
	if req.Email != "" {
		existedUser, err = s.userDao.GetByEmail(ctx, req.Email)
		if err != nil {
			return nil, err
		}
	}
	if existedUser == nil && req.Phone != "" {
		existedUser, err = s.userDao.GetByPhone(ctx, req.Phone)
		if err != nil {
			return nil, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var targetUserID int64
	if existedUser != nil {
		// 已存在，检查是否已是成员
		existedM, err := s.membershipDao.GetByAccountAndUser(ctx, req.AccountID, existedUser.ID)
		if err != nil {
			return nil, err
		}
		if existedM != nil && existedM.Status != "left" {
			return nil, errs.ErrMembershipExists
		}
		targetUserID = existedUser.ID
	} else {
		// 不存在则建 User + Credential + Personal Account
		hash, err := password.Hash(req.Password)
		if err != nil {
			return nil, err
		}
		newUserID := snowflake.Next()
		displayName := req.Nickname
		if displayName == "" {
			displayName = req.RealName
		}
		if displayName == "" {
			if req.Email != "" {
				displayName = strings.Split(req.Email, "@")[0]
			} else {
				displayName = "user-" + req.Phone
			}
		}

		userPO := &dao.UserPO{
			ID:             newUserID,
			Nickname:       nullString(displayName),
			RealName:       nullString(req.RealName),
			Status:         "active",
			RegisterSource: "org_created",
			MustChangePwd:  true,
		}
		if req.Email != "" {
			userPO.Email = nullString(req.Email)
		}
		if req.Phone != "" {
			userPO.Phone = nullString(req.Phone)
		}
		if err := s.userDao.Insert(ctx, tx, userPO); err != nil {
			return nil, err
		}
		if err := s.credentialDao.Insert(ctx, tx, &dao.UserCredentialPO{
			UserID:       newUserID,
			PasswordHash: hash,
			PasswordAlgo: s.cfg.Account.PasswordAlgo,
		}); err != nil {
			return nil, err
		}
		// 子用户也建一份 Personal Account（与主流 SaaS 一致）
		personalID := snowflake.Next()
		personalMID := snowflake.Next()
		personalPO := &dao.AccountPO{
			ID:          personalID,
			Type:        "personal",
			Code:        buildAccountCode("personal"),
			Name:        displayName,
			OwnerUserID: newUserID,
			Status:      "active",
			Plan:        "free",
			MemberLimit: 1,
		}
		if err := s.accountDao.Insert(ctx, tx, personalPO); err != nil {
			return nil, err
		}
		if err := s.membershipDao.Insert(ctx, tx, &dao.MembershipPO{
			ID:          personalMID,
			AccountID:   personalID,
			UserID:      newUserID,
			Role:        "owner",
			Status:      "active",
			DisplayName: nullString(displayName),
			JoinSource:  "owner_init",
		}); err != nil {
			return nil, err
		}

		targetUserID = newUserID
	}

	// 建目标企业的 membership
	targetMID := snowflake.Next()
	displayName := req.Nickname
	if displayName == "" {
		displayName = req.RealName
	}
	mpo := &dao.MembershipPO{
		ID:          targetMID,
		AccountID:   req.AccountID,
		UserID:      targetUserID,
		Role:        role,
		Status:      "active",
		DisplayName: nullString(displayName),
		JoinSource:  "org_created",
		InvitedBy:   nullInt64(req.CurrentUserID),
	}
	if err := s.membershipDao.Insert(ctx, tx, mpo); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	v := toMembershipView(mpo)
	return &v, nil
}

// ListMembersReq 列成员入参。
type ListMembersReq struct {
	CurrentUserID int64
	AccountID     int64
	Offset        int
	Limit         int
}

// ListMembers 列出企业成员（含 User 展开）。
func (s *MemberService) ListMembers(ctx context.Context, req ListMembersReq) ([]MembershipView, error) {
	if req.CurrentUserID == 0 {
		return nil, errs.ErrUnauth
	}
	// 只有本 account 的活跃成员才能查
	m, err := s.membershipDao.GetByAccountAndUser(ctx, req.AccountID, req.CurrentUserID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.Status != "active" {
		return nil, errs.ErrForbidden
	}

	pos, err := s.membershipDao.ListByAccount(ctx, req.AccountID, req.Offset, req.Limit)
	if err != nil {
		return nil, err
	}
	out := make([]MembershipView, 0, len(pos))
	for i := range pos {
		v := toMembershipView(&pos[i])
		// 展开 User
		u, err := s.userDao.GetByID(ctx, pos[i].UserID)
		if err == nil && u != nil {
			uv := toUserView(u)
			v.User = &uv
		}
		out = append(out, v)
	}
	return out, nil
}

// RemoveMemberReq 移除成员入参。
type RemoveMemberReq struct {
	CurrentUserID int64
	AccountID     int64
	MembershipID  int64
}

// RemoveMember 移除成员（软移除：status → left）。owner 不能被移除。
func (s *MemberService) RemoveMember(ctx context.Context, req RemoveMemberReq) error {
	if req.CurrentUserID == 0 {
		return errs.ErrUnauth
	}
	curM, err := s.membershipDao.GetByAccountAndUser(ctx, req.AccountID, req.CurrentUserID)
	if err != nil {
		return err
	}
	if curM == nil || curM.Status != "active" || (curM.Role != "owner" && curM.Role != "admin") {
		return errs.ErrForbidden
	}
	target, err := s.membershipDao.GetByID(ctx, req.MembershipID)
	if err != nil {
		return err
	}
	if target == nil || target.AccountID != req.AccountID {
		return errs.ErrMembershipNotFound
	}
	if target.Role == "owner" {
		return errs.ErrCannotRemoveOwner
	}
	return s.membershipDao.MarkLeft(ctx, target.ID)
}
