// view.go 定义 service 层对外暴露的领域视图（DTO），以及一些 sql.Null* 的构造 helper。
//
// 目的：
//   - 与 dao.PO 解耦（PO 里到处是 sql.Null* / TINYINT，不适合直接给 controller）；
//   - 提供 JSON 友好的字段格式，controller 层可以直接返回。
package service

import (
	"database/sql"
	"time"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/dao"
)

// UserView 用户领域视图。
type UserView struct {
	ID              int64      `json:"id"`
	Username        string     `json:"username,omitempty"`
	Email           string     `json:"email,omitempty"`
	EmailVerified   bool       `json:"email_verified"`
	Phone           string     `json:"phone,omitempty"`
	PhoneVerified   bool       `json:"phone_verified"`
	Nickname        string     `json:"nickname,omitempty"`
	RealName        string     `json:"real_name,omitempty"`
	AvatarURL       string     `json:"avatar_url,omitempty"`
	Gender          int        `json:"gender"`
	Status          string     `json:"status"`
	MustChangePwd   bool       `json:"must_change_pwd"`
	IsPlatformAdmin bool       `json:"is_platform_admin"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreateTime      time.Time  `json:"create_time"`
}

// AccountView 账户领域视图。
type AccountView struct {
	ID          int64     `json:"id"`
	Type        string    `json:"type"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	OwnerUserID int64     `json:"owner_user_id"`
	LogoURL     string    `json:"logo_url,omitempty"`
	Status      string    `json:"status"`
	Verified    bool      `json:"verified"`
	Plan        string    `json:"plan"`
	MemberLimit int       `json:"member_limit"`
	CreateTime  time.Time `json:"create_time"`
}

// MembershipView 成员领域视图（可选带 User / Account 展开信息，controller 层按需返回）。
type MembershipView struct {
	ID          int64      `json:"id"`
	AccountID   int64      `json:"account_id"`
	UserID      int64      `json:"user_id"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	DisplayName string     `json:"display_name,omitempty"`
	JoinSource  string     `json:"join_source"`
	JoinTime    time.Time  `json:"join_time"`
	LeaveTime   *time.Time `json:"leave_time,omitempty"`
	InvitedBy   int64      `json:"invited_by,omitempty"`

	// 可选展开
	User    *UserView    `json:"user,omitempty"`
	Account *AccountView `json:"account,omitempty"`
}

// InvitationView 邀请视图。
type InvitationView struct {
	ID           int64     `json:"id"`
	AccountID    int64     `json:"account_id"`
	InviterID    int64     `json:"inviter_id"`
	InviteeEmail string    `json:"invitee_email,omitempty"`
	InviteePhone string    `json:"invitee_phone,omitempty"`
	Role         string    `json:"role"`
	Token        string    `json:"token"`
	Status       string    `json:"status"`
	ExpireAt     time.Time `json:"expire_at"`
	CreateTime   time.Time `json:"create_time"`
}

// ===== PO -> View 映射 =====

func toUserView(po *dao.UserPO) UserView {
	if po == nil {
		return UserView{}
	}
	v := UserView{
		ID:              po.ID,
		Username:        nullStr(po.Username),
		Email:           nullStr(po.Email),
		EmailVerified:   po.EmailVerified,
		Phone:           nullStr(po.Phone),
		PhoneVerified:   po.PhoneVerified,
		Nickname:        nullStr(po.Nickname),
		RealName:        nullStr(po.RealName),
		AvatarURL:       nullStr(po.AvatarURL),
		Gender:          po.Gender,
		Status:          po.Status,
		MustChangePwd:   po.MustChangePwd,
		IsPlatformAdmin: po.IsPlatformAdmin,
		CreateTime:      po.CreateTime,
	}
	if po.LastLoginAt.Valid {
		t := po.LastLoginAt.Time
		v.LastLoginAt = &t
	}
	return v
}

func toAccountView(po *dao.AccountPO) AccountView {
	if po == nil {
		return AccountView{}
	}
	return AccountView{
		ID:          po.ID,
		Type:        po.Type,
		Code:        po.Code,
		Name:        po.Name,
		OwnerUserID: po.OwnerUserID,
		LogoURL:     nullStr(po.LogoURL),
		Status:      po.Status,
		Verified:    po.Verified,
		Plan:        po.Plan,
		MemberLimit: po.MemberLimit,
		CreateTime:  po.CreateTime,
	}
}

func toMembershipView(po *dao.MembershipPO) MembershipView {
	if po == nil {
		return MembershipView{}
	}
	v := MembershipView{
		ID:          po.ID,
		AccountID:   po.AccountID,
		UserID:      po.UserID,
		Role:        po.Role,
		Status:      po.Status,
		DisplayName: nullStr(po.DisplayName),
		JoinSource:  po.JoinSource,
		JoinTime:    po.JoinTime,
	}
	if po.LeaveTime.Valid {
		t := po.LeaveTime.Time
		v.LeaveTime = &t
	}
	if po.InvitedBy.Valid {
		v.InvitedBy = po.InvitedBy.Int64
	}
	return v
}

func toInvitationView(po *dao.InvitationPO) InvitationView {
	if po == nil {
		return InvitationView{}
	}
	return InvitationView{
		ID:           po.ID,
		AccountID:    po.AccountID,
		InviterID:    po.InviterID,
		InviteeEmail: nullStr(po.InviteeEmail),
		InviteePhone: nullStr(po.InviteePhone),
		Role:         po.Role,
		Token:        po.Token,
		Status:       po.Status,
		ExpireAt:     po.ExpireAt,
		CreateTime:   po.CreateTime,
	}
}

// ===== sql.Null* helpers =====

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullStr(n sql.NullString) string {
	if !n.Valid {
		return ""
	}
	return n.String
}

func nullInt64(v int64) sql.NullInt64 {
	if v == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: v, Valid: true}
}
