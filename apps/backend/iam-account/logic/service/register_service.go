// register_service.go 提供个人注册、企业注册两种注册链路。
//
// 个人注册：邮箱/手机 + 验证码 + 密码 → 创建 User + Credential + Personal Account + Membership。
// 企业注册：登录态调用，为当前 User 创建 Organization Account + OrgProfile + Owner Membership。
//
// 所有多表写入都放在事务里，事务失败整体回滚。
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

// RegisterService 注册服务。
type RegisterService struct {
	db             *sql.DB
	cfg            *config.Detail
	verifyCode     *VerifyCodeService
	userDao        dao.UserDao
	credentialDao  dao.UserCredentialDao
	accountDao     dao.AccountDao
	orgProfileDao  dao.AccountOrgProfileDao
	membershipDao  dao.MembershipDao
}

// NewRegisterService 构造。
func NewRegisterService(
	db *sql.DB, cfg *config.Detail, vc *VerifyCodeService,
	u dao.UserDao, c dao.UserCredentialDao,
	a dao.AccountDao, op dao.AccountOrgProfileDao, m dao.MembershipDao,
) *RegisterService {
	return &RegisterService{
		db: db, cfg: cfg, verifyCode: vc,
		userDao: u, credentialDao: c,
		accountDao: a, orgProfileDao: op, membershipDao: m,
	}
}

// RegisterPersonalReq 个人注册入参。
type RegisterPersonalReq struct {
	Email    string
	Phone    string
	Code     string // 验证码
	Password string
	Nickname string
}

// RegisterResp 注册出参（个人 / 企业通用）。
type RegisterResp struct {
	UserID    int64  `json:"user_id"`
	AccountID int64  `json:"account_id"`
	Account   AccountView `json:"account"`
}

// RegisterPersonal 个人注册。
func (s *RegisterService) RegisterPersonal(ctx context.Context, req RegisterPersonalReq) (*RegisterResp, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)
	req.Password = strings.TrimSpace(req.Password)
	req.Nickname = strings.TrimSpace(req.Nickname)

	if req.Email == "" && req.Phone == "" {
		return nil, errs.ErrBadRequest.WithMessage("email or phone is required")
	}
	if len(req.Password) < 8 {
		return nil, errs.ErrBadRequest.WithMessage("password must be at least 8 chars")
	}
	if req.Code == "" {
		return nil, errs.ErrBadRequest.WithMessage("verify code is required")
	}

	// 1. 校验验证码
	target := req.Email
	if target == "" {
		target = req.Phone
	}
	if err := s.verifyCode.Verify(ctx, target, "register", req.Code); err != nil {
		return nil, err
	}

	// 2. 冲突检测
	if req.Email != "" {
		u, err := s.userDao.GetByEmail(ctx, req.Email)
		if err != nil {
			return nil, err
		}
		if u != nil {
			return nil, errs.ErrEmailAlreadyUsed
		}
	}
	if req.Phone != "" {
		u, err := s.userDao.GetByPhone(ctx, req.Phone)
		if err != nil {
			return nil, err
		}
		if u != nil {
			return nil, errs.ErrPhoneAlreadyUsed
		}
	}

	// 3. hash 密码
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	// 4. 事务：user + credential + account(personal) + membership(owner)
	userID := snowflake.Next()
	accountID := snowflake.Next()
	membershipID := snowflake.Next()

	displayName := req.Nickname
	if displayName == "" {
		if req.Email != "" {
			displayName = strings.Split(req.Email, "@")[0]
		} else {
			displayName = "user-" + req.Phone
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// user
	userPO := &dao.UserPO{
		ID:             userID,
		Nickname:       nullString(displayName),
		Status:         "active",
		RegisterSource: "self",
	}
	if req.Email != "" {
		userPO.Email = nullString(req.Email)
		userPO.EmailVerified = true
	}
	if req.Phone != "" {
		userPO.Phone = nullString(req.Phone)
		userPO.PhoneVerified = true
	}
	if err := s.userDao.Insert(ctx, tx, userPO); err != nil {
		return nil, err
	}
	// credential
	if err := s.credentialDao.Insert(ctx, tx, &dao.UserCredentialPO{
		UserID:       userID,
		PasswordHash: hash,
		PasswordAlgo: s.cfg.Account.PasswordAlgo,
	}); err != nil {
		return nil, err
	}
	// account (personal)
	accPO := &dao.AccountPO{
		ID:          accountID,
		Type:        "personal",
		Code:        buildAccountCode("personal"),
		Name:        displayName,
		OwnerUserID: userID,
		Status:      "active",
		Plan:        "free",
		MemberLimit: 1,
	}
	if err := s.accountDao.Insert(ctx, tx, accPO); err != nil {
		return nil, err
	}
	// membership (owner)
	if err := s.membershipDao.Insert(ctx, tx, &dao.MembershipPO{
		ID:          membershipID,
		AccountID:   accountID,
		UserID:      userID,
		Role:        "owner",
		Status:      "active",
		DisplayName: nullString(displayName),
		JoinSource:  "owner_init",
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &RegisterResp{
		UserID:    userID,
		AccountID: accountID,
		Account:   toAccountView(accPO),
	}, nil
}

// RegisterOrganizationReq 企业注册入参。登录态调用（携带 currentUserID）。
type RegisterOrganizationReq struct {
	CurrentUserID   int64
	Name            string // 企业显示名
	LegalName       string // 企业注册名
	UnifiedCreditNo string
	Industry        string
	Scale           string
	Province        string
	City            string
	Address         string
	ContactEmail    string
	ContactPhone    string
}

// RegisterOrganization 创建企业账户。
func (s *RegisterService) RegisterOrganization(ctx context.Context, req RegisterOrganizationReq) (*RegisterResp, error) {
	if req.CurrentUserID == 0 {
		return nil, errs.ErrUnauth
	}
	req.Name = strings.TrimSpace(req.Name)
	req.LegalName = strings.TrimSpace(req.LegalName)
	if req.Name == "" {
		return nil, errs.ErrBadRequest.WithMessage("name is required")
	}
	if req.LegalName == "" {
		req.LegalName = req.Name
	}

	// 确认当前 user 存在
	u, err := s.userDao.GetByID(ctx, req.CurrentUserID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errs.ErrUserNotFound
	}

	accountID := snowflake.Next()
	membershipID := snowflake.Next()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	accPO := &dao.AccountPO{
		ID:          accountID,
		Type:        "organization",
		Code:        buildAccountCode("org"),
		Name:        req.Name,
		OwnerUserID: req.CurrentUserID,
		Status:      "active",
		Plan:        "free",
		MemberLimit: 20,
	}
	if err := s.accountDao.Insert(ctx, tx, accPO); err != nil {
		return nil, err
	}
	if err := s.orgProfileDao.Insert(ctx, tx, &dao.AccountOrgProfilePO{
		AccountID:       accountID,
		LegalName:       req.LegalName,
		UnifiedCreditNo: nullString(req.UnifiedCreditNo),
		Industry:        nullString(req.Industry),
		Scale:           nullString(req.Scale),
		Province:        nullString(req.Province),
		City:            nullString(req.City),
		Address:         nullString(req.Address),
		ContactEmail:    nullString(req.ContactEmail),
		ContactPhone:    nullString(req.ContactPhone),
	}); err != nil {
		return nil, err
	}
	if err := s.membershipDao.Insert(ctx, tx, &dao.MembershipPO{
		ID:          membershipID,
		AccountID:   accountID,
		UserID:      req.CurrentUserID,
		Role:        "owner",
		Status:      "active",
		DisplayName: u.Nickname,
		JoinSource:  "owner_init",
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &RegisterResp{
		UserID:    req.CurrentUserID,
		AccountID: accountID,
		Account:   toAccountView(accPO),
	}, nil
}
