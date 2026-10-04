// verify_code_service.go 是"发送/校验验证码"服务。
//
// 简化实现：
//   - 发送：本模块不真的发邮件/短信，只把明文验证码写日志，供开发环境使用。
//     生产环境可注入 Sender 接口。
//   - 存储：t_verify_code 记录 code_hash（sha256），校验时用 hash 比对。
//   - 防爆破：单条验证码 try_count 达到上限（默认 5）则强制标记 used。
//   - 频率限制：60 秒内同 target+scene 不允许再次发送。
package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/errs"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/snowflake"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/config"
	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/dao"
)

// VerifyCodeService 验证码服务。
type VerifyCodeService struct {
	dao dao.VerifyCodeDao
	cfg *config.Detail
}

// NewVerifyCodeService 构造。
func NewVerifyCodeService(d dao.VerifyCodeDao, cfg *config.Detail) *VerifyCodeService {
	return &VerifyCodeService{dao: d, cfg: cfg}
}

// SendVerifyCodeReq 发送验证码入参。
type SendVerifyCodeReq struct {
	Channel  string // email | sms
	Target   string
	Scene    string // register | reset_pwd | bind | invite | mfa
	ClientIP string
}

// SendVerifyCodeResp 发送验证码出参。
type SendVerifyCodeResp struct {
	ExpireInSec int `json:"expire_in_sec"`
	// DebugCode 仅本地/开发环境使用；由服务判断（生产不返）。
	// 当前实现总是返回，用于开发前后端联调；上线前请去掉。
	DebugCode string `json:"debug_code,omitempty"`
}

// SendCode 发送验证码。
func (s *VerifyCodeService) SendCode(ctx context.Context, req SendVerifyCodeReq) (*SendVerifyCodeResp, error) {
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	req.Target = strings.TrimSpace(req.Target)
	req.Scene = strings.TrimSpace(req.Scene)
	if req.Channel != "email" && req.Channel != "sms" {
		return nil, errs.ErrBadRequest.WithMessage("channel must be email or sms")
	}
	if req.Target == "" {
		return nil, errs.ErrBadRequest.WithMessage("target is required")
	}
	if req.Scene == "" {
		return nil, errs.ErrBadRequest.WithMessage("scene is required")
	}

	// 频率限制：60 秒内不允许重复发送。
	last, err := s.dao.LatestCreatedAt(ctx, req.Target, req.Scene)
	if err != nil {
		return nil, err
	}
	if !last.IsZero() && time.Since(last) < 60*time.Second {
		return nil, errs.ErrVerifyCodeTooFast
	}

	code := randomDigits(6)
	ttl := time.Duration(s.cfg.Account.VerifyCodeTTLSeconds) * time.Second

	var clientIP *string
	if req.ClientIP != "" {
		ip := req.ClientIP
		clientIP = &ip
	}

	po := &dao.VerifyCodePO{
		ID:       snowflake.Next(),
		Channel:  req.Channel,
		Target:   req.Target,
		Scene:    req.Scene,
		CodeHash: sha256Hex(code),
		ExpireAt: time.Now().Add(ttl),
	}
	if clientIP != nil {
		po.ClientIP.String = *clientIP
		po.ClientIP.Valid = true
	}
	if err := s.dao.Insert(ctx, po); err != nil {
		return nil, err
	}

	// 开发/本地环境：把验证码打到日志。生产环境请替换为真实发送器。
	log.Printf("[iam-account][verify-code] channel=%s target=%s scene=%s code=%s ttl=%s",
		req.Channel, req.Target, req.Scene, code, ttl)

	return &SendVerifyCodeResp{
		ExpireInSec: s.cfg.Account.VerifyCodeTTLSeconds,
		DebugCode:   code,
	}, nil
}

// Verify 校验验证码。校验通过后标记 used。
//
// 语义：
//   - 无最新可用记录：ErrVerifyCodeInvalid
//   - hash 不匹配：try_count+1；若达上限则强制标记 used 视为消耗
func (s *VerifyCodeService) Verify(ctx context.Context, target, scene, code string) error {
	po, err := s.dao.GetLatestActive(ctx, target, scene)
	if err != nil {
		return err
	}
	if po == nil {
		return errs.ErrVerifyCodeInvalid
	}
	if po.CodeHash != sha256Hex(code) {
		_ = s.dao.IncrementTry(ctx, po.ID)
		if po.TryCount+1 >= s.cfg.Account.VerifyCodeMaxRetries {
			_ = s.dao.MarkUsed(ctx, po.ID)
		}
		return errs.ErrVerifyCodeInvalid
	}
	if err := s.dao.MarkUsed(ctx, po.ID); err != nil {
		return errors.Join(err, errs.ErrInternal)
	}
	return nil
}
