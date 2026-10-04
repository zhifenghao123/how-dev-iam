// access_log.go 提供"全量请求/响应"访问日志中间件。
//
// 与 gin 内置 LoggerWithConfig 的差异：
//   - gin 内置 Logger 只记请求行 + 状态码 + 耗时；
//   - 本中间件以 JSON 一行的形式追加记录请求体 / 响应体 / 客户端 IP /
//     request_id / query / content-type 等，便于事后审计与排障。
//
// 设计要点：
//  1. 请求体可重复读取：先 io.ReadAll 读完 Body，再以 io.NopCloser 塞回
//     c.Request.Body，handler 仍可正常解析。
//  2. 响应体捕获：包一层 gin.ResponseWriter，写入时同步复制到 buffer。
//  3. 体积保护：单边最多 8 KiB；超出截断并带 "...(truncated)" 标记。
//  4. 类型保护：仅文本类 Content-Type 才记录原文；二进制只记长度。
//  5. 输出位置：默认写入 logging.AccessWriter()。
package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/zhifenghao123/how-dev-iam/iam-authz/logic/common/logging"

	"github.com/gin-gonic/gin"
)

// 访问日志的体积保护阈值。
const accessLogMaxBodyBytes = 8 * 1024 // 8 KiB

// AccessLogConfig 访问日志中间件的可选配置。
type AccessLogConfig struct {
	SkipPaths    []string
	MaxBodyBytes int
	Writer       io.Writer
}

// defaultAccessLogSkipPaths 默认跳过的低价值路径。
var defaultAccessLogSkipPaths = []string{
	"/healthCheck/status",
	"/metrics",
	"/favicon.ico",
}

// AccessLog 返回一个把每次请求记录为单行 JSON 的中间件。
func AccessLog(cfg AccessLogConfig) gin.HandlerFunc {
	skip := make(map[string]struct{})
	paths := cfg.SkipPaths
	if len(paths) == 0 {
		paths = defaultAccessLogSkipPaths
	}
	for _, p := range paths {
		skip[p] = struct{}{}
	}
	maxBody := cfg.MaxBodyBytes
	if maxBody <= 0 {
		maxBody = accessLogMaxBodyBytes
	}

	return func(c *gin.Context) {
		if _, ok := skip[c.Request.URL.Path]; ok {
			c.Next()
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/swagger/") {
			c.Next()
			return
		}

		start := time.Now()

		// 读取并重置请求体
		var reqBody []byte
		if c.Request.Body != nil {
			if b, err := io.ReadAll(c.Request.Body); err == nil {
				reqBody = b
				c.Request.Body = io.NopCloser(bytes.NewReader(b))
			}
		}

		// 包一层 ResponseWriter，捕获响应体
		bw := &bodyCaptureWriter{
			ResponseWriter: c.Writer,
			buf:            bytes.NewBuffer(nil),
			limit:          maxBody,
		}
		c.Writer = bw

		c.Next()

		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0

		entry := map[string]any{
			"time":       start.Format(time.RFC3339Nano),
			"client_ip":  c.ClientIP(),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"query":      c.Request.URL.RawQuery,
			"status":     c.Writer.Status(),
			"latency_ms": latencyMs,
			"request":    renderBody(c.ContentType(), reqBody, maxBody),
			"response":   renderBody(bw.Header().Get("Content-Type"), bw.buf.Bytes(), maxBody),
		}
		if rid, ok := c.Get(CtxKeyRequestID); ok {
			entry["request_id"] = rid
		}

		line, err := json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(gin.DefaultWriter, "[access] marshal failed: %v\n", err)
			return
		}
		w := cfg.Writer
		if w == nil {
			w = logging.AccessWriter()
		}
		_, _ = w.Write(append(line, '\n'))
	}
}

// bodyCaptureWriter 包一层 gin.ResponseWriter，把响应体复制到 buf。
type bodyCaptureWriter struct {
	gin.ResponseWriter
	buf   *bytes.Buffer
	limit int
}

func (w *bodyCaptureWriter) Write(p []byte) (int, error) {
	if remain := w.limit - w.buf.Len(); remain > 0 {
		if len(p) <= remain {
			w.buf.Write(p)
		} else {
			w.buf.Write(p[:remain])
		}
	}
	return w.ResponseWriter.Write(p)
}

func (w *bodyCaptureWriter) WriteString(s string) (int, error) {
	if remain := w.limit - w.buf.Len(); remain > 0 {
		if len(s) <= remain {
			w.buf.WriteString(s)
		} else {
			w.buf.WriteString(s[:remain])
		}
	}
	return w.ResponseWriter.WriteString(s)
}

// renderBody 根据 content-type 决定如何在日志中展示 body。
func renderBody(contentType string, body []byte, limit int) string {
	if len(body) == 0 {
		return ""
	}
	if !isTextualContentType(contentType) {
		return fmt.Sprintf("<binary %d bytes>", len(body))
	}
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit]) + "...(truncated)"
}

// isTextualContentType 判断 Content-Type 是否属于"可读文本"。
func isTextualContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return true
	}
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch {
	case strings.HasPrefix(ct, "text/"):
		return true
	case ct == "application/json", strings.HasSuffix(ct, "+json"):
		return true
	case ct == "application/xml", strings.HasSuffix(ct, "+xml"):
		return true
	case ct == "application/x-www-form-urlencoded":
		return true
	case ct == "application/javascript", ct == "application/ecmascript":
		return true
	default:
		return false
	}
}
