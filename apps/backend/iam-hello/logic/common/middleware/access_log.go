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
//  3. 体积保护：单边最多 8 KiB；超出截断并带 "...(truncated)" 标记，
//     防止单次大上传/大下载把日志撑爆。
//  4. 类型保护：仅文本类 Content-Type 才记录原文；二进制（multipart、
//     image、octet-stream 等）只记长度，避免污染日志。
//  5. 输出位置：默认写入 logging.AccessWriter()（独立的访问日志
//     文件，例如 log/<service>-req.log），与启动 / 业务日志彻底分离；
//     也可通过 AccessLogConfig.Writer 显式指定。
package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic/common/logging"

	"github.com/gin-gonic/gin"
)

// 访问日志的体积保护阈值。
const accessLogMaxBodyBytes = 8 * 1024 // 8 KiB

// AccessLogConfig 访问日志中间件的可选配置。
type AccessLogConfig struct {
	// SkipPaths 不记录访问日志的精确路径列表（与 gin LoggerConfig.SkipPaths 同语义）。
	// 为空时使用 defaultAccessLogSkipPaths。
	SkipPaths []string
	// MaxBodyBytes 单边（请求 / 响应）允许记录的最大字节数；<=0 时使用默认 8 KiB。
	MaxBodyBytes int
	// Writer 访问日志输出目标；nil 时运行期从 logging.AccessWriter() 获取，
	// 保证在 logging.Setup 后才能拿到文件句柄，且在 logging.Close 后自动降级为 Discard。
	Writer io.Writer
}

// defaultAccessLogSkipPaths 默认跳过的低价值路径。
var defaultAccessLogSkipPaths = []string{
	"/healthCheck/status",
	"/metrics",
	"/favicon.ico",
}

// AccessLog 返回一个把每次请求记录为单行 JSON 的中间件。
//
// 用法：engine.Use(middleware.AccessLog(middleware.AccessLogConfig{}))
//
// 字段示意（单行 JSON）：
//
//	{
//	  "time":"2026-06-13T22:30:13.123456789+08:00",
//	  "client_ip":"127.0.0.1",
//	  "method":"GET",
//	  "path":"/api/authn/supportTypes",
//	  "query":"",
//	  "status":200,
//	  "latency_ms":0.551,
//	  "request_id":"...",
//	  "request":"",
//	  "response":"{\"types\":[...]}"
//	}
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
		// 1) 跳过名单：以 path 为准（不含 query），与 gin Logger 行为一致。
		if _, ok := skip[c.Request.URL.Path]; ok {
			c.Next()
			return
		}
		// swagger 静态资源整体跳过
		if strings.HasPrefix(c.Request.URL.Path, "/swagger/") {
			c.Next()
			return
		}

		start := time.Now()

		// 2) 读取并重置请求体，确保后续 handler 仍能正常解析。
		var reqBody []byte
		if c.Request.Body != nil {
			if b, err := io.ReadAll(c.Request.Body); err == nil {
				reqBody = b
				c.Request.Body = io.NopCloser(bytes.NewReader(b))
			}
		}

		// 3) 包一层 ResponseWriter，捕获响应体。
		bw := &bodyCaptureWriter{
			ResponseWriter: c.Writer,
			buf:            bytes.NewBuffer(nil),
			limit:          maxBody,
		}
		c.Writer = bw

		// 4) 真正执行后续 handler
		c.Next()

		// 5) 组装并打一行 JSON
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

		// 用 json.Marshal 而非 Encoder：避免 Encoder 自动追加换行后我们再加一次。
		line, err := json.Marshal(entry)
		if err != nil {
			// 极端情况下序列化失败，降级为简单文本，确保至少有一行可读痕迹。
			fmt.Fprintf(gin.DefaultWriter, "[access] marshal failed: %v\n", err)
			return
		}
		// 主动追加换行；运行期解析 writer（避免中间件构造时 logging 尚未 Setup）。
		w := cfg.Writer
		if w == nil {
			w = logging.AccessWriter()
		}
		_, _ = w.Write(append(line, '\n'))
	}
}

// bodyCaptureWriter 包一层 gin.ResponseWriter，把响应体复制到 buf。
//
// 仅捕获到 limit 字节，超出部分不再追加进 buf（但 Write 本身全量透传，
// 不影响真实响应）。这样既保住业务正确性，又避免大响应撑爆日志缓冲区。
type bodyCaptureWriter struct {
	gin.ResponseWriter
	buf   *bytes.Buffer
	limit int
}

// Write 透传写入到底层 ResponseWriter，并在 limit 内同步复制到 buf。
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

// WriteString 与 Write 同语义；gin 部分场景会调到 WriteString。
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
//
//   - 文本类（json / xml / form / text/*）：原文，超出 limit 截断；
//   - 二进制类：仅记 "<binary N bytes>"，避免乱码；
//   - 空 body：返回空串。
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
//
// 空 content-type 视为文本（GET 请求大概率无 body，或简单文本响应）。
func isTextualContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return true
	}
	// 截掉 charset 等参数
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
