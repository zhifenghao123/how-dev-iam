// Package config 实现脚手架的配置层。
//
// 设计原则：
//
//  1. 三段式构造：applyDefaults -> parseFromFile -> validate -> applyFallbacks。
//  2. 启动期 fail-fast：任何字段非法都直接返回 error，避免运行时崩溃。
//  3. 全局只读访问：通过 SetGlobal/Global，跨包获取配置不必再注入。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/zhifenghao123/how-dev-iam/iam-authz/logic/constant"
)

// Detail 是服务的完整配置。
type Detail struct {
	LogLevel    string `json:"log_level"`
	HTTPPort    int    `json:"http_port"`
	GRPCPort    int    `json:"grpc_port,omitempty"` // 0 表示不启用 gRPC
	MetricsPort int    `json:"metrics_port"`
	ServiceName string `json:"service_name"`

	// EnableSwagger 是否暴露 /swagger/*any 接口文档。
	// 当前脚手架未接入 swag，仅保留字段占位，运行期忽略。
	EnableSwagger bool `json:"enable_swagger"`

	Log   LogConfig   `json:"log"`
	Trace TraceConfig `json:"trace"`
}

// LogConfig 日志输出配置。
type LogConfig struct {
	Dir      string `json:"dir"`
	File     string `json:"file"`
	ReqFile  string `json:"req_file"`
	ToStdout bool   `json:"to_stdout"`
}

// TraceConfig OTel 链路追踪配置。
type TraceConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	Name     string `json:"name"`
}

// applyDefaults 设置默认值。仅在反序列化前调用，会被 JSON 中的非零值覆盖。
func (d *Detail) applyDefaults() {
	d.LogLevel = "info"
	d.HTTPPort = 8000
	d.MetricsPort = 9090
	d.ServiceName = constant.ServiceName
	d.EnableSwagger = false
	d.Log.Dir = "log"
	d.Log.File = fmt.Sprintf("%s.log", constant.ServiceName)
	d.Log.ReqFile = fmt.Sprintf("%s-req.log", constant.ServiceName)
	d.Log.ToStdout = true
	d.Trace.Name = constant.ServiceName
}

// parseFromFile 从 JSON 文件读取并反序列化。
func (d *Detail) parseFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %q: %w", path, err)
	}
	if err := json.Unmarshal(data, d); err != nil {
		return fmt.Errorf("unmarshal config %q: %w", path, err)
	}
	return nil
}

// validate 校验关键字段合法性。
func (d *Detail) validate() error {
	if d.HTTPPort < 1 || d.HTTPPort > 65535 {
		return fmt.Errorf("invalid http_port: %d", d.HTTPPort)
	}
	if d.MetricsPort < 1 || d.MetricsPort > 65535 {
		return fmt.Errorf("invalid metrics_port: %d", d.MetricsPort)
	}
	if d.GRPCPort != 0 && (d.GRPCPort < 1 || d.GRPCPort > 65535) {
		return fmt.Errorf("invalid grpc_port: %d", d.GRPCPort)
	}
	if d.HTTPPort == d.MetricsPort {
		return fmt.Errorf("http_port and metrics_port must differ: both=%d", d.HTTPPort)
	}
	if d.GRPCPort != 0 && (d.GRPCPort == d.HTTPPort || d.GRPCPort == d.MetricsPort) {
		return fmt.Errorf("grpc_port conflicts with http/metrics: grpc=%d http=%d metrics=%d",
			d.GRPCPort, d.HTTPPort, d.MetricsPort)
	}
	return nil
}

// applyFallbacks 反序列化后才能判断空值的兜底逻辑放这里，避免被 JSON 覆盖。
func (d *Detail) applyFallbacks() {
	if d.ServiceName == "" {
		d.ServiceName = constant.ServiceName
	}
	if d.Trace.Name == "" {
		d.Trace.Name = d.ServiceName
	}
	if d.Log.Dir == "" {
		d.Log.Dir = "log"
	}
	if d.Log.File == "" {
		d.Log.File = d.ServiceName + ".log"
	}
	if d.Log.ReqFile == "" {
		d.Log.ReqFile = d.ServiceName + "-req.log"
	}
}

// New 加载配置。脚手架的唯一入口。
func New(path string) (*Detail, error) {
	var d Detail
	d.applyDefaults()
	if err := d.parseFromFile(path); err != nil {
		return nil, err
	}
	if err := d.validate(); err != nil {
		return nil, err
	}
	d.applyFallbacks()
	SetGlobal(&d)
	return &d, nil
}

// ===== 全局只读访问 =====

var (
	mu     sync.RWMutex
	global *Detail
)

// SetGlobal 设置全局配置。
func SetGlobal(c *Detail) {
	mu.Lock()
	defer mu.Unlock()
	global = c
}

// Global 获取全局配置。
func Global() *Detail {
	mu.RLock()
	defer mu.RUnlock()
	return global
}
