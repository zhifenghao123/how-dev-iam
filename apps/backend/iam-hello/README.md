# iam-hello

`how-dev-iam` 仓库下的最小可运行示例服务，用于演示"新服务脚手架"的分层与装配约定。

## 目录结构

```
iam-hello/
├── main.go                       # 仅装配 + 生命周期编排
├── go.mod
├── README.md
├── conf/
│   └── config.json               # 默认配置文件
├── docs/                         # 接口文档：由 `swag init -g main.go -o ./docs` 生成
│   ├── docs.go                   # （首次接入为占位文件，swag init 后被覆盖）
│   ├── swagger.json
│   └── swagger.yaml
├── httptest/                     # JetBrains/REST Client 兼容的接口用例
│   ├── authn_manage_controller.http  # authn 控制器接口用例
│   ├── authz_manage_controller.http  # authz 控制器接口用例
│   └── ops.http                  # health / metrics / pprof 等基础设施用例
├── log/                          # 运行时生成：默认日志输出目录
└── logic/                        # 业务核心代码
    ├── app.go                    # package logic：聚合根 App，装配链 + lifecycle
    ├── common/
    │   ├── logging/
    │   │   └── logging.go        # 日志双通道初始化：主日志 + 请求访问日志（独立文件）
    │   └── middleware/
    │       ├── request_id.go     # X-Request-Id 透传中间件
    │       └── access_log.go     # 请求访问日志中间件：单行 JSON，写入独立文件
    ├── config/
    │   └── config.go             # 三段式配置：applyDefaults -> parseFromFile -> validate -> applyFallbacks
    ├── route/                    # 路由注册层
    │   └── router.go             # Module interface + Register / RegisterMetrics / RegisterSwagger
    ├── controller/               # handler 层（XxxController struct + 方法 + RegisterTo）
    │   ├── authn_manage_controller.go  # AuthnManageController：SupportType
    │   └── authz_manage_controller.go  # AuthzManageController：SupportType
    ├── service/                  # 业务层（XxxService struct + 方法）
    │   ├── authn_manage_service.go     # AuthnManageService：SupportTypes
    │   └── authz_manage_service.go     # AuthzManageService：SupportTypes
    ├── dao/                      # 数据访问层：dao 接口 + SQLite 实现
    │   ├── sqlite.go                   # 共享内存 SQLite（file::memory + cache=shared）+ SQL 脚本执行器
    │   ├── authn_manage_dao.go         # AuthnTypeDao + AuthnTypeSQLiteDao
    │   └── authz_manage_dao.go         # AuthzTypeDao + AuthzTypeSQLiteDao
    ├── db/                       # SQL 资源（//go:embed 嵌入二进制）
    │   ├── embed.go                    # DDLFS / DMLFS / ReadDDL / ReadDML
    │   ├── ddl/                        # 建表语句（一表一文件，文件名同表名）
    │   │   ├── t_authn_type.sql
    │   │   └── t_authz_type.sql
    │   └── dml/                        # 种子数据（INSERT OR IGNORE，幂等）
    │       ├── t_authn_type.sql
    │       └── t_authz_type.sql
    └── remote_call/              # 外部系统调用占位目录（按需扩展 RPC / HTTP client）
```

## 分层约定

- **controller**：每个领域一个 `XxxController` struct，handler 作为 struct 方法；
  通过实现 `route.Module` 接口的 `RegisterTo(group)` 把自身路由挂到 `/api` 下。
- **service**：每个领域一个 `XxxService` struct，业务方法作为 struct 方法；
  依赖（dao、外部 client）由构造函数注入到 struct 字段中，方便单测 mock。
- **dao**：`interface + struct 实现`。service 仅依赖 dao 接口，不直接 import DB driver。当前内置 SQLite 实现（`modernc.org/sqlite`，纯 Go，零 CGO），运行在 `file::memory + cache=shared` 模式：进程内多次读写共享同一份内存库，进程退出即销毁。
- **db（资源）**：`logic/db/` 下用 `//go:embed` 把 `ddl/*.sql` 与 `dml/*.sql` 编译进二进制，`embed.go` 暴露 `DDLFS()` / `DMLFS()` 给 dao。物理 SQL 文件保留为可读文本，DBA 可直接拷到生产库执行。
- **route**：定义 `Module` interface，对具体业务零感知；`Register(engine, modules...)` 用可变参数收集所有 controller。
- **logic.App**：聚合根，只与 controller 对话；各 controller 模块提供 `NewDefault*Controller` 便利构造函数，内部完成 `db(资源) -> dao -> service -> controller` 默认装配。App 把 controller 作为 Module 注入路由。

## 启动

在仓库根目录下执行：

```bash
cd iam-hello
go mod tidy
go run . -config ./conf/config.json
```

`-config` 默认值就是 `./conf/config.json`，本地无需显式指定。

预期日志：

```
[iam-hello] logger initialized: file=log/iam-hello.log req_file=log/iam-hello-req.log to_stdout=true
[iam-hello] pprof listening on :6060
[iam-hello] HTTP server starting on :8000
[iam-hello] Metrics server starting on :9090
```

## 日志

默认在启动目录下创建 `log/`，并以追加模式同时写入两个文件（**双通道分离**）：

```
iam-hello/log/iam-hello.log        # 主日志：业务 log.Printf + gin 启动/error/Recovery
iam-hello/log/iam-hello-req.log    # 请求访问日志：每次请求一行 JSON（method/path/status/latency_ms/request/response/...）
```

- 可在 [conf/config.json](./conf/config.json) 的 `log` 段中调整：
  - `dir`：日志目录（相对启动工作目录，也可填绝对路径）。
  - `file`：主日志文件名（默认 `<service_name>.log`）。
  - `req_file`：请求访问日志文件名（默认 `<service_name>-req.log`）。
  - `to_stdout`：主日志是否同时在控制台输出；本地开发建议 `true`，生产建议 `false`。**请求访问日志始终只写文件**，不污染 stdout。
- 主日志接管范围：标准库 `log.Printf` + gin 访问日志（`gin.LoggerWithConfig`）+ gin panic 信息（`gin.Recovery`）。
- 请求访问日志由 [logic/common/middleware/access_log.go](./logic/common/middleware/access_log.go) 产生，writer 由 [logic/common/logging/logging.go](./logic/common/logging/logging.go) 通过包级 `AccessWriter()` 暴露，避免 logging 反向依赖 middleware。
- 默认跳过低价值路径：`/healthCheck/status`、`/metrics`、`/favicon.ico`。
- 追加模式，重启不会丢失历史日志；后续如需轮换，可在 [logic/common/logging/logging.go](./logic/common/logging/logging.go) 中接入 lumberjack 等。

## 验证接口

推荐直接用 IDE 打开 [httptest/authn_manage_controller.http](./httptest/authn_manage_controller.http) / [httptest/authz_manage_controller.http](./httptest/authz_manage_controller.http) / [httptest/ops.http](./httptest/ops.http) 一键发送（JetBrains IDE 与 VS Code REST Client 均原生支持）。

也可用 curl：

```bash
# 1. 健康检查（被 LoggerWithConfig 跳过日志）
curl http://127.0.0.1:8000/healthCheck/status
# {"service":"iam-hello","status":"ok"}

# 2. 业务接口：支持的认证方式列表
curl http://127.0.0.1:8000/api/authn/supportTypes
# {"types":[{"code":"password","name":"用户名密码",...}, ...]}

# 3. 业务接口：支持的认证方式列表 + Request-ID 透传
curl -H 'X-Request-Id: my-trace-001' \
'http://127.0.0.1:8000/api/authn/supportTypes' -i
# 响应头会回写 X-Request-Id: my-trace-001

# 4. 业务接口：支持的授权方式列表
curl http://127.0.0.1:8000/api/authz/supportTypes
# {"types":[{"code":"rbac","name":"RBAC",...}, ...]}

# 5. metrics 独立端口
curl http://127.0.0.1:9090/metrics

# 6. pprof
curl http://127.0.0.1:6060/debug/pprof/

# 7. 接口文档（Swagger UI，仅在 enable_swagger=true 时可访问）
open http://127.0.0.1:8000/swagger/index.html
```

## 接口文档（gin-swagger）

本项目集成了 [`gin-swagger`](https://github.com/swaggo/gin-swagger)，接口文档采用“注解驱动”方式：
在 controller handler 上写 `// @Summary ...` 等注解 -> `swag init` 生成 `docs/` 包
-> 运行时由 `gin-swagger` 提供的 handler 接管 `/swagger/*any`。

### 分层安放

| 关注点 | 位置 | 说明 |
|---|---|---|
| `@title` / `@host` / `@BasePath` 总注解 | [main.go](./main.go) | swag 默认从入口扫描 |
| 接口注解 `@Summary` / `@Param` / `@Router` | controller 函数上 | 依然“近邻”接口 |
| `/swagger/*any` 路由 | [logic/route/router.go](./logic/route/router.go) `RegisterSwagger` | 与 health/metrics 同为基础设施路由 |
| 启用开关 | `enable_swagger`（[conf/config.json](./conf/config.json)）| 生产环境建议设为 `false` |
| 生成产物 | [docs/](./docs/) | `swag init` 覆盖；首次接入为占位文件 |

“business controller 不感知 swagger”：swagger 被看作与 health/metrics 同类的基础设施路由，
由 `route` 包独立提供 `RegisterSwagger`，`App.initHTTPEngine` 根据配置开关按需调用。

### 安装 swag CLI 与生成文档

```bash
# 1) 一次性：安装 swag CLI（建议 ≥ v1.16，与运行时 swaggo/swag 库版本对齐）
go install github.com/swaggo/swag/cmd/swag@v1.16.3

# 2) 拉取运行时依赖（首次接入后执行）
cd iam-hello
go mod tidy

# 3) 生成/更新文档。每当调整了接口注解都需重跑。
swag init -g main.go -o ./docs

# 4) 启动服务并访问
open http://127.0.0.1:8000/swagger/index.html
```

> **版本一致性提醒**：`swag init` CLI 与 `go.mod` 中的 `github.com/swaggo/swag`
> 必须保持同一大版本。CLI ≥ v1.16 生成的 `docs.go` 会使用 `LeftDelim` / `RightDelim`
> 字段，旧的 `swag` 库（v1.8.x）没有该字段，会编译失败：
> `unknown field LeftDelim in struct literal of type "github.com/swaggo/swag".Spec`。
> 本仓库已固定到 `v1.16.3`，团队成员请勿随意降级。

> 提示：首次接入时仓库内的 `docs/docs.go` 是一个最小可编译的**占位包**，
> 仅为了让未跑 `swag init` 的源码仓库也能 `go build` 通过。
> 一旦运行过 `swag init`，该占位会被生成产物覆盖。

### 在 controller 上加注解（示例）

参考 [logic/controller/authn_manage_controller.go](./logic/controller/authn_manage_controller.go)，每个 handler 上加入：

```go
// @Summary  支持的认证方式列表
// @Tags     authn
// @Produce  json
// @Success  200  {object}  map[string]interface{}
// @Router   /authn/supportTypes [get]
func (ac *AuthnManageController) SupportType(c *gin.Context) { ... }
```

`@Router` 写到 `BasePath`（`/api`）之后的部分。

### 生产环境安全策略

- 在 [conf/config.json](./conf/config.json) 中将 `enable_swagger` 设为 `false`，
  `App.initHTTPEngine` 会跳过 `RegisterSwagger`，`/swagger/*` 返回 404。
- 若仅限内网访问，可结合 nginx / 网关层进一步限制。

## 优雅关闭验证

```bash
# 终端 A：启动
go run .

# 终端 B：发起一个长耗时请求（注意：示例没有长流式接口，可用 sleep 模拟）
# 这里仅演示 Ctrl+C 关闭流程的日志：
# [iam-hello] received interrupt, graceful shutdown...
# [iam-hello] step1: stop accepting new requests
# [iam-hello] step2: wait active requests, current=0
# [iam-hello] step3: close app resources
# [iam-hello] shutdown completed
# [iam-hello] exited
```

## 验收清单（脚手架方案对齐）

| 设计要点 | 实现位置 |
|---|---|
| main 仅装配 | [main.go](./main.go) |
| 三段式配置 + validate | [logic/config/config.go](./logic/config/config.go) `New` |
| 端口三分离（HTTP/Metrics/pprof） | [main.go](./main.go) + `Detail.HTTPPort/MetricsPort` |
| `App` 聚合根 + controller 装配 + 生命周期 | [logic/app.go](./logic/app.go) |
| 配置全局只读访问 | [logic/config/config.go](./logic/config/config.go) `Global` / `SetGlobal` |
| `Module` interface + `Register(engine, modules...)` | [logic/route/router.go](./logic/route/router.go) |
| `RegisterMetrics(engine) error` | 同上 |
| 中间件顺序：Logger(skip 健康检查) → Recovery → RequestID → AccessLog → 业务 | [logic/route/router.go](./logic/route/router.go) `Register` |
| `/healthCheck/status` | 同上 |
| `/swagger/*any` 接口文档（可开关） | [logic/route/router.go](./logic/route/router.go) `RegisterSwagger` + `Detail.EnableSwagger` |
| 请求访问日志（单行 JSON / 独立文件） | [logic/common/middleware/access_log.go](./logic/common/middleware/access_log.go) + [logic/common/logging/logging.go](./logic/common/logging/logging.go) |
| controller struct + 方法 + `RegisterTo` | [logic/controller/authn_manage_controller.go](./logic/controller/authn_manage_controller.go) |
| service struct + 方法 + 构造函数注入 | [logic/service/authn_manage_service.go](./logic/service/authn_manage_service.go) |
| dao 接口 + SQLite struct 实现 | [logic/dao/authn_manage_dao.go](./logic/dao/authn_manage_dao.go) / [logic/dao/authz_manage_dao.go](./logic/dao/authz_manage_dao.go) / [logic/dao/sqlite.go](./logic/dao/sqlite.go) |
| SQL 资源嵌入（//go:embed） | [logic/db/embed.go](./logic/db/embed.go) + [logic/db/ddl/](./logic/db/ddl) + [logic/db/dml/](./logic/db/dml) |
| 多 server goroutine + errCh | [main.go](./main.go) |
| 优雅关闭：Shutdown → WaitForActiveRequests → Close | [main.go](./main.go) `gracefulShutdown` + [logic/app.go](./logic/app.go) |
| 活跃请求计数 | `App.activeReq atomic.Int64` |

## 新增一个业务子模块（机械化步骤）

1. **dao**（如需持久化）：
   1. 在 [logic/db/ddl/](./logic/db/ddl) 与 [logic/db/dml/](./logic/db/dml) 下分别新建 `<table>.sql`（建表语句 / 种子数据），文件名与表名同名；建表语句建议带 `IF NOT EXISTS`，种子数据建议用 `INSERT OR IGNORE` 保证幂等。
   2. 在 [logic/dao/](./logic/dao/) 下新建 `<module>_dao.go`：
      - 定义 `XxxPO` 持久化对象 + `XxxDao` interface；
      - 提供 struct 实现 `XxxYyyDao`（`Yyy` = `SQLite` / `MySQL` / `Redis` / `Memory` 等）；
      - 构造函数 `NewXxxYyyDao(ctx, db, ddlFS, dmlFS)` 内部通过 `dao.ExecSQLScript` 执行 DDL / DML 脚本，启动期完成建表与种子写入。
2. **service**：在 [logic/service/](./logic/service/) 下新建 `<module>.go`：
   - 定义 `XxxService` struct，依赖（dao、外部 client）作为字段；
   - 构造函数 `NewXxxService(deps...)` 显式注入；
   - 业务方法挂在 struct 上，不依赖 gin / HTTP。
3. **controller**：在 [logic/controller/](./logic/controller/) 下新建 `<module>_controller.go`：
   - 定义 `XxxController` struct，字段持有 `*XxxService`；
   - 构造函数 `NewXxxController(svc)` 显式注入；
   - 额外提供 `NewDefaultXxxController()` 便利构造函数，内部调用 `service.NewDefaultXxxService()`，
     把默认装配链封装在模块内部；
   - 同时在 service 包中提供 `NewDefaultXxxService()`，内部用 `dao.OpenSharedMemorySQLite("<namespace>")` + `dao.NewXxxYyyDao(ctx, db, dbres.DDLFS(), dbres.DMLFS())` 完成默认装配；
   - 实现 `RegisterTo(group *gin.RouterGroup)`（即 `route.Module`），把路由挂到 `/api/<module>` 下；
   - 各 handler 作为 struct 方法，仅做入参提取 + 调用 service + 响应封装.
4. **logic.App 装配**：在 [logic/app.go](./logic/app.go) 的 `controllers()` 方法返回切片中追加一行：
   ```go
   func (a *App) controllers() []route.Module {
       return []route.Module{
           controller.NewDefaultAuthnManageController(),
           controller.NewDefaultAuthzManageController(),
           controller.NewDefaultXxxController(), // ← 新增这一行
       }
   }
   ```
   App struct 与 `NewFromConfig` 都无需任何改动；App 不持有任何 controller 字段，
   也不感知 service / dao 的具体类型。
5. **route 包不需要改动**——它对具体业务零感知。
6. **接口文档（可选）**：在新 controller 的各 handler 上补充 swag 注解
   （`@Summary` / `@Tags` / `@Param` / `@Success` / `@Router` 等），随后执行：
   ```bash
   swag init -g main.go -o ./docs
   ```
   即可在 Swagger UI 中看到新接口；`docs/` 包由 swag 生成，业务代码无需手动修改。
7. 完成。业务代码从未 `import` `App`，零循环依赖。
