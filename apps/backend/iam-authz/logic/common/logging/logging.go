// Package logging 负责把标准库 log 与 gin 默认 Writer
// 重定向到磁盘文件，供本服务统一使用。
//
// 设计要点：
//  1. 初始化时机：必须在 config.New 之后、任何 log.Printf 与
//     gin.New 之前调用，确保后续日志都进文件。
//  2. 双写：当 ToStdout=true 时，主日志同时写文件和标准输出，
//     便于本地调试；访问日志为避免噪音始终只写文件。
//  3. 双通道：
//     - 主文件（cfg.Log.File）：业务 / 启动 / gin 内置访问日志；
//     - 访问日志文件（cfg.Log.ReqFile）：仅 access log JSON 行。
//  4. 资源回收：返回 *Logger，由调用方在进程退出前 Close()。
//  5. 中间件解耦：通过包级 AccessWriter() 暴露 access writer，
//     middleware 包按需获取，避免 logging 反向依赖 middleware。
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/zhifenghao123/how-dev-iam/iam-authz/logic/config"
	"github.com/zhifenghao123/how-dev-iam/iam-authz/logic/constant"

	"github.com/gin-gonic/gin"
)

// Logger 持有日志文件句柄，便于优雅退出时关闭。
type Logger struct {
	file    *os.File // 主日志文件
	reqFile *os.File // 访问日志文件
	path    string   // 主日志文件路径
	reqPath string   // 访问日志文件路径
}

// Path 返回当前主日志文件路径。
func (l *Logger) Path() string { return l.path }

// ReqPath 返回当前访问日志文件路径。
func (l *Logger) ReqPath() string { return l.reqPath }

// Close 关闭底层文件句柄。多次调用是安全的。
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	var firstErr error
	if l.file != nil {
		if err := l.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		l.file = nil
	}
	if l.reqFile != nil {
		if err := l.reqFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		l.reqFile = nil
	}
	accessWriter.Store((*writerHolder)(nil))
	return firstErr
}

// ===== 全局 access writer =====

type writerHolder struct{ w io.Writer }

// accessWriter 持有当前进程的访问日志 writer。
//
// 用 atomic.Pointer 而非互斥锁：写入仅在 Setup / Close 触发，
// 中间件每次请求都要读取，无锁路径优先。
var accessWriter atomic.Pointer[writerHolder]

// AccessWriter 返回当前访问日志 writer。
//
// 若 Setup 尚未调用或 Close 已执行，返回 io.Discard。
func AccessWriter() io.Writer {
	if h := accessWriter.Load(); h != nil && h.w != nil {
		return h.w
	}
	return io.Discard
}

// Setup 根据配置创建日志目录、打开日志文件，并接管标准库 log 与 gin 默认 writer。
func Setup(cfg *config.Detail) (*Logger, error) {
	if cfg == nil {
		return nil, fmt.Errorf("logging.Setup: cfg is nil")
	}

	dir := cfg.Log.Dir
	if dir == "" {
		dir = "log"
	}
	mainName := cfg.Log.File
	if mainName == "" {
		mainName = cfg.ServiceName + ".log"
	}
	reqName := cfg.Log.ReqFile
	if reqName == "" {
		reqName = cfg.ServiceName + "-req.log"
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("logging.Setup: mkdir %q: %w", dir, err)
	}

	// 1) 主日志文件
	mainPath := filepath.Join(dir, mainName)
	mainFile, err := os.OpenFile(mainPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("logging.Setup: open %q: %w", mainPath, err)
	}

	var mainW io.Writer = mainFile
	if cfg.Log.ToStdout {
		mainW = io.MultiWriter(mainFile, os.Stdout)
	}

	log.SetOutput(mainW)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	gin.DefaultWriter = mainW
	gin.DefaultErrorWriter = mainW

	// 2) 访问日志文件
	reqPath := filepath.Join(dir, reqName)
	reqFile, err := os.OpenFile(reqPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		_ = mainFile.Close()
		return nil, fmt.Errorf("logging.Setup: open %q: %w", reqPath, err)
	}
	accessWriter.Store(&writerHolder{w: reqFile})

	log.Printf("[%s] logger initialized: file=%s req_file=%s to_stdout=%v",
		constant.ServiceName, mainPath, reqPath, cfg.Log.ToStdout)
	return &Logger{
		file:    mainFile,
		reqFile: reqFile,
		path:    mainPath,
		reqPath: reqPath,
	}, nil
}
