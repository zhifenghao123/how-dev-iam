// Package service 是 iam-account 的业务逻辑层。
//
// 本文件放各 service 共用的小工具函数，避免每个文件都写一份。
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	mrand "math/rand"
	"strings"
	"time"
)

// randomDigits 生成 n 位随机数字字符串（左侧补零）。
//
// 用途：邮箱/短信验证码；使用 crypto/rand，安全性足够。
func randomDigits(n int) string {
	if n <= 0 {
		return ""
	}
	// 最大值 10^n
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	v, err := rand.Int(rand.Reader, max)
	if err != nil {
		// 极端情况下退回伪随机，仅作兜底。
		v = big.NewInt(int64(mrand.Intn(1_000_000)))
	}
	return fmt.Sprintf("%0*d", n, v.Int64())
}

// randomToken 生成 hex 编码的 n 字节随机 token（供邀请令牌使用）。
func randomToken(nBytes int) (string, error) {
	if nBytes <= 0 {
		nBytes = 24
	}
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// sha256Hex 计算 SHA-256 hex 摘要。验证码 hash 用。
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// buildAccountCode 基于时间戳 + 后缀构造账户 code（人类可读、全局大概率唯一）。
//
// 例：personal-20260808120001-a1b2c3、org-20260808120001-a1b2c3
func buildAccountCode(prefix string) string {
	stamp := time.Now().Format("20060102150405")
	suffix, _ := randomToken(3)
	return fmt.Sprintf("%s-%s-%s", prefix, stamp, strings.ToLower(suffix))
}
