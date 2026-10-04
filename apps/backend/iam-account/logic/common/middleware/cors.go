// cors.go：CORS 中间件，最小实现，支持前后端分离本地联调。
//
// 生产环境应把 AllowOrigins 收敛到白名单，并根据需要开启 Credentials。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig CORS 参数。
type CORSConfig struct {
	// AllowOrigins 允许的源，为空或包含 "*" 视为全部允许。
	AllowOrigins []string
	// AllowMethods 允许的方法，默认常见方法。
	AllowMethods []string
	// AllowHeaders 允许的请求头。
	AllowHeaders []string
	// AllowCredentials 是否允许携带凭据（Cookie/HTTPAuth）。
	AllowCredentials bool
	// MaxAgeSec 预检请求缓存时长（秒）。
	MaxAgeSec int
}

// CORS 返回一个 Gin 中间件。
func CORS(cfg CORSConfig) gin.HandlerFunc {
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowHeaders) == 0 {
		cfg.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-Id", "X-Request-Id"}
	}
	if cfg.MaxAgeSec <= 0 {
		cfg.MaxAgeSec = 600
	}

	allowAll := len(cfg.AllowOrigins) == 0
	for _, o := range cfg.AllowOrigins {
		if o == "*" {
			allowAll = true
			break
		}
	}
	allowSet := make(map[string]struct{}, len(cfg.AllowOrigins))
	for _, o := range cfg.AllowOrigins {
		allowSet[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if allowAll {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			} else if _, ok := allowSet[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ","))
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ","))
			if cfg.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			c.Header("Access-Control-Max-Age", intToStr(cfg.MaxAgeSec))
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func intToStr(n int) string {
	// 简单实现避免 strconv 依赖噪声
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 12)
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
