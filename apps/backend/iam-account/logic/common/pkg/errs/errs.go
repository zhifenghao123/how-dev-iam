// Package errs 定义 iam-account 的业务错误码。
//
// controller 层根据 Err.HTTPStatus 返回状态码，前端根据 Code 做 i18n / 分支跳转。
package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// Err 是带 code / http status 的业务错误。
type Err struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

// Error 实现 error。
func (e *Err) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// WithMessage 返回一个 message 被替换的 Err 副本，code / status 不变。
func (e *Err) WithMessage(msg string) *Err {
	return &Err{Code: e.Code, Message: msg, HTTPStatus: e.HTTPStatus}
}

// As 从任意 error 提取 *Err；未包装时返回 (nil, false)。
func As(err error) (*Err, bool) {
	var e *Err
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// 预定义错误码（组织成组，便于扩展）。
var (
	// 通用
	ErrInternal   = &Err{Code: "INTERNAL", Message: "internal server error", HTTPStatus: http.StatusInternalServerError}
	ErrBadRequest = &Err{Code: "BAD_REQUEST", Message: "bad request", HTTPStatus: http.StatusBadRequest}
	ErrUnauth     = &Err{Code: "UNAUTHORIZED", Message: "unauthorized", HTTPStatus: http.StatusUnauthorized}
	ErrForbidden  = &Err{Code: "FORBIDDEN", Message: "forbidden", HTTPStatus: http.StatusForbidden}
	ErrNotFound   = &Err{Code: "NOT_FOUND", Message: "not found", HTTPStatus: http.StatusNotFound}
	ErrConflict   = &Err{Code: "CONFLICT", Message: "conflict", HTTPStatus: http.StatusConflict}

	// 验证码
	ErrVerifyCodeInvalid = &Err{Code: "VERIFY_CODE_INVALID", Message: "verify code invalid or expired", HTTPStatus: http.StatusBadRequest}
	ErrVerifyCodeTooFast = &Err{Code: "VERIFY_CODE_TOO_FAST", Message: "please retry later", HTTPStatus: http.StatusTooManyRequests}

	// 注册 / 用户
	ErrEmailAlreadyUsed = &Err{Code: "EMAIL_ALREADY_USED", Message: "email already registered", HTTPStatus: http.StatusConflict}
	ErrPhoneAlreadyUsed = &Err{Code: "PHONE_ALREADY_USED", Message: "phone already registered", HTTPStatus: http.StatusConflict}
	ErrUserNotFound     = &Err{Code: "USER_NOT_FOUND", Message: "user not found", HTTPStatus: http.StatusNotFound}
	ErrPasswordWrong    = &Err{Code: "PASSWORD_WRONG", Message: "password wrong", HTTPStatus: http.StatusUnauthorized}

	// 账户 / 成员
	ErrAccountNotFound     = &Err{Code: "ACCOUNT_NOT_FOUND", Message: "account not found", HTTPStatus: http.StatusNotFound}
	ErrAccountNotOrg       = &Err{Code: "ACCOUNT_NOT_ORG", Message: "operation only allowed on organization account", HTTPStatus: http.StatusBadRequest}
	ErrMemberLimitExceed   = &Err{Code: "MEMBER_LIMIT_EXCEED", Message: "member limit exceeded", HTTPStatus: http.StatusBadRequest}
	ErrMembershipExists    = &Err{Code: "MEMBERSHIP_EXISTS", Message: "user is already a member", HTTPStatus: http.StatusConflict}
	ErrMembershipNotFound  = &Err{Code: "MEMBERSHIP_NOT_FOUND", Message: "membership not found", HTTPStatus: http.StatusNotFound}
	ErrCannotRemoveOwner   = &Err{Code: "CANNOT_REMOVE_OWNER", Message: "cannot remove owner", HTTPStatus: http.StatusBadRequest}

	// 邀请
	ErrInvitationInvalid  = &Err{Code: "INVITATION_INVALID", Message: "invitation token invalid", HTTPStatus: http.StatusBadRequest}
	ErrInvitationExpired  = &Err{Code: "INVITATION_EXPIRED", Message: "invitation expired", HTTPStatus: http.StatusBadRequest}
	ErrInvitationConsumed = &Err{Code: "INVITATION_CONSUMED", Message: "invitation already used", HTTPStatus: http.StatusBadRequest}
)
