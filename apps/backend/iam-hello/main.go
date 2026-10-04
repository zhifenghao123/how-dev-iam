// iam-hello 是按"新服务脚手架"方案落地的最小可运行示例。
//
// 启动：
//
//	cd iam-hello
//	go mod tidy
//	go run . -config ./conf/config.json
//
// 验证：
//
//	curl http://127.0.0.1:8000/healthCheck/status
//	curl http://127.0.0.1:8000/api/authn/supportTypes
//	curl http://127.0.0.1:8000/api/authz/supportTypes
//	curl http://127.0.0.1:9090/metrics
//
// main.go 仅做装配 + 生命周期编排，零业务逻辑。
//
// @title           iam-hello API
// @version         1.0
// @description     iam-hello 服务接口文档（脚手架示例）。
// @description     仅在 config 中 enable_swagger=true 时暴露 /swagger/*any。
// @host            127.0.0.1:8000
// @BasePath        /api
// @schemes         http
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof" // pprof 监听在独立端口 :6060
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic"
	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic/common/logging"
	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic/config"
	"github.com/zhifenghao123/how-dev-iam/iam-hello/logic/constant"
)

var (
	configPath      = flag.String("config", "./conf/config.json", "config file path")
	shutdownTimeout = flag.Duration("shutdown-timeout", 30*time.Second, "graceful shutdown timeout")
	pprofAddr       = flag.String("pprof-addr", ":6060", "pprof listen address (empty to disable)")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 0) 先加载配置 + 初始化日志，确保后续所有 log.Printf / gin 访问日志
	//    都写到文件。这里是唯一允许在 logger 接管前写 stderr 的位置。
	cfg, err := config.New(*configPath)
	if err != nil {
		log.Fatalf("[%s] load config failed: %v", constant.HelloServiceName, err)
	}
	logger, err := logging.Setup(cfg)
	if err != nil {
		log.Fatalf("[%s] init logger failed: %v", constant.HelloServiceName, err)
	}
	defer logger.Close()

	// 1) pprof（独立端口，独立 goroutine，不影响主流程）
	if *pprofAddr != "" {
		go func() {
			log.Printf("[%s] pprof listening on %s", constant.HelloServiceName, *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Printf("[%s] pprof exit: %v", constant.HelloServiceName, err)
			}
		}()
	}

	// 2) 装配 App（内部只与 controller 对话；service / dao 装配封装在各 controller 模块内）
	svr, err := logic.NewFromConfig(cfg)
	if err != nil {
		log.Fatalf("[%s] logic.NewFromConfig failed: %v", constant.HelloServiceName, err)
	}
	defer svr.Close()

	// 3) 启动多个 server，错误统一汇聚到 errCh
	errCh := make(chan error, 2)
	go func() {
		if err := svr.ListenHTTP(ctx); err != nil {
			errCh <- err
		}
	}()
	go func() {
		if err := svr.ListenMetrics(ctx); err != nil {
			errCh <- err
		}
	}()

	// 4) 信号 / 错误监听
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("[%s] received %v, graceful shutdown...", constant.HelloServiceName, sig)
	case err := <-errCh:
		log.Fatalf("[%s] server startup error: %v", constant.HelloServiceName, err)
	}
	signal.Stop(sigCh)

	// 5) 优雅关闭
	gracefulShutdown(ctx, svr, *shutdownTimeout)
	log.Printf("[%s] exited", constant.HelloServiceName)
}

// gracefulShutdown 执行 N 步关闭：
//  1. Shutdown：停止接收新连接（http.Server.Shutdown 会等已完成的连接）
//  2. WaitForActiveRequests：等待长流式请求完成
//  3. Close：释放 App 持有资源
//
// 任一步骤超时则强制退出。
func gracefulShutdown(ctx context.Context, svr *logic.App, timeout time.Duration) {
	shutdownCtx, c := context.WithTimeout(context.Background(), timeout)
	defer c()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Step 1: 停止接收新请求
		log.Printf("[%s] step1: stop accepting new requests", constant.HelloServiceName)
		svr.Shutdown(shutdownCtx)
		// Step 2: 等活跃请求完成
		log.Printf("[%s] step2: wait active requests, current=%d", constant.HelloServiceName, svr.ActiveRequestCount())
		if ok := svr.WaitForActiveRequests(shutdownCtx); !ok {
			log.Printf("[%s] step2: timeout, %d active requests remain", constant.HelloServiceName, svr.ActiveRequestCount())
		}
		// Step 3: 释放资源
		log.Printf("[%s] step3: close app resources", constant.HelloServiceName)
		svr.Close()
	}()

	select {
	case <-done:
		log.Printf("[%s] shutdown completed", constant.HelloServiceName)
	case <-shutdownCtx.Done():
		log.Printf("[%s] shutdown timeout, force exit; active=%d", constant.HelloServiceName, svr.ActiveRequestCount())
	}
}
