// iam-authn 是账号服务的最小可运行脚手架，
// 仅包含一个 HealthCheck 接口，业务与 DB 访问尚未接入。
//
// 启动：
//
//	cd iam-authn
//	go mod tidy
//	go run . -config ./conf/config.json
//
// 验证：
//
//	curl http://127.0.0.1:8002/healthCheck/status
//	curl http://127.0.0.1:8002/api/healthCheck/ping
//	curl http://127.0.0.1:9092/metrics
//
// main.go 仅做装配 + 生命周期编排，零业务逻辑。
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof" // pprof 监听在独立端口 :6062
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhifenghao123/how-dev-iam/iam-authn/logic"
	"github.com/zhifenghao123/how-dev-iam/iam-authn/logic/common/logging"
	"github.com/zhifenghao123/how-dev-iam/iam-authn/logic/config"
	"github.com/zhifenghao123/how-dev-iam/iam-authn/logic/constant"
)

var (
	configPath      = flag.String("config", "./conf/config.json", "config file path")
	shutdownTimeout = flag.Duration("shutdown-timeout", 30*time.Second, "graceful shutdown timeout")
	pprofAddr       = flag.String("pprof-addr", ":6062", "pprof listen address (empty to disable)")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 0) 先加载配置 + 初始化日志
	cfg, err := config.New(*configPath)
	if err != nil {
		log.Fatalf("[%s] load config failed: %v", constant.ServiceName, err)
	}
	logger, err := logging.Setup(cfg)
	if err != nil {
		log.Fatalf("[%s] init logger failed: %v", constant.ServiceName, err)
	}
	defer logger.Close()

	// 1) pprof（独立端口，独立 goroutine）
	if *pprofAddr != "" {
		go func() {
			log.Printf("[%s] pprof listening on %s", constant.ServiceName, *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Printf("[%s] pprof exit: %v", constant.ServiceName, err)
			}
		}()
	}

	// 2) 装配 App
	svr, err := logic.NewFromConfig(cfg)
	if err != nil {
		log.Fatalf("[%s] logic.NewFromConfig failed: %v", constant.ServiceName, err)
	}
	defer svr.Close()

	// 3) 启动 HTTP / Metrics server
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
		log.Printf("[%s] received %v, graceful shutdown...", constant.ServiceName, sig)
	case err := <-errCh:
		log.Fatalf("[%s] server startup error: %v", constant.ServiceName, err)
	}
	signal.Stop(sigCh)

	// 5) 优雅关闭
	gracefulShutdown(ctx, svr, *shutdownTimeout)
	log.Printf("[%s] exited", constant.ServiceName)
}

// gracefulShutdown 执行三步优雅关闭：
//  1. Shutdown：停止接收新连接
//  2. WaitForActiveRequests：等活跃请求完成
//  3. Close：释放 App 资源
func gracefulShutdown(ctx context.Context, svr *logic.App, timeout time.Duration) {
	shutdownCtx, c := context.WithTimeout(context.Background(), timeout)
	defer c()

	done := make(chan struct{})
	go func() {
		defer close(done)
		log.Printf("[%s] step1: stop accepting new requests", constant.ServiceName)
		svr.Shutdown(shutdownCtx)
		log.Printf("[%s] step2: wait active requests, current=%d", constant.ServiceName, svr.ActiveRequestCount())
		if ok := svr.WaitForActiveRequests(shutdownCtx); !ok {
			log.Printf("[%s] step2: timeout, %d active requests remain", constant.ServiceName, svr.ActiveRequestCount())
		}
		log.Printf("[%s] step3: close app resources", constant.ServiceName)
		svr.Close()
	}()

	select {
	case <-done:
		log.Printf("[%s] shutdown completed", constant.ServiceName)
	case <-shutdownCtx.Done():
		log.Printf("[%s] shutdown timeout, force exit; active=%d", constant.ServiceName, svr.ActiveRequestCount())
	}
}
