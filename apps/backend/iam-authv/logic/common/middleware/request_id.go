// Package middleware 提供脚手架内置的通用中间件。
package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const (
	// HeaderRequestID 是全链路 Request ID 的请求头名。
	HeaderRequestID = "X-Request-Id"
	// CtxKeyRequestID 是 gin context 中存放 request id 的 key。
	CtxKeyRequestID = "request_id"
)

// RequestID 生成或透传请求 ID，并写回响应头。
//
// 上游已有 X-Request-Id 时透传，缺失时本服务生成一个 16 字节随机串。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(HeaderRequestID)
		if rid == "" {
			rid = newRandomID()
		}
		c.Set(CtxKeyRequestID, rid)
		c.Writer.Header().Set(HeaderRequestID, rid)
		c.Next()
	}
}

// newRandomID 用 crypto/rand 生成 32 位 16 进制随机串，避免引入 uuid 依赖。
func newRandomID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(buf[:])
}
