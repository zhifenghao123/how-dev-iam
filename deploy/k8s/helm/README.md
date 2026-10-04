# how-dev-iam Helm Charts

Chart 目录结构与参数参考。**部署指南和一键脚本请见** [`../../README.md`](../../README.md)。

## 目录结构

```
deploy/k8s/helm/
├── k8s-infra/           # 基础公共组件 umbrella chart
│   ├── charts/          # (git 忽略) helm dep update 后放官方 ingress-nginx
│   └── templates/
│       └── iam-ingress.yaml  # iam 平台对外 Ingress
├── middleware/          # 中间件 umbrella chart
│   └── charts/
│       ├── mysql/       # MySQL 8.0（模板直接放在 templates/ 下，无子目录）
│       └── redis/       # Redis 7
├── init-jobs/           # 一次性初始化 Job（helm hook）
│   └── ddl/iam-account/ # iam-account 服务的建表 SQL
└── iam/                 # 业务 umbrella chart（不再包含 Ingress）
    └── charts/
        ├── iam-account/ # 账号服务
        ├── iam-front/   # 前端（nginx + SPA + /api 反代）
        ├── iam-authn/   # 预留（暂未实现）
        ├── iam-authv/   # 预留（暂未实现）
        └── iam-authz/   # 预留（暂未实现）
```

## Chart 关键参数

### `k8s-infra` (基础公共组件)

| Key | 默认 | 说明 |
|---|---|---|
| `ingress-nginx.enabled` | `true` | 是否安装 ingress-nginx controller |
| `ingress-nginx.controller.replicaCount` | `1` | 本地开发单副本即可 |
| `ingress-nginx.controller.service.type` | `LoadBalancer` | 可改为 `NodePort`（80 被占时） |
| `iamIngress.enabled` | `true` | 是否创建 iam 平台 Ingress 资源 |
| `iamIngress.namespace` | `iam` | Ingress 部署到哪个 ns（与 iam-front Service 同 ns） |
| `iamIngress.host` | `iam.local` | 对外域名 |
| `iamIngress.className` | `nginx` | IngressClass 名 |
| `iamIngress.frontServiceName` | `iam-front` | 后端 Service 名 |
| `iamIngress.frontServicePort` | `80` | 后端 Service 端口 |
| `iamIngress.tls.enabled` | `false` | 是否启用 TLS |

**使用前**必须先执行 `helm dependency update deploy/k8s/helm/k8s-infra`（`deploy-infra.sh` 自动做了这一步）。

### `iam` (业务 umbrella)

**注意**：Ingress 已迁到 `k8s-infra`，本 chart 不再包含 Ingress 相关字段。

| Key | 默认 | 说明 |
|---|---|---|
| `global.image.registry` | `host.docker.internal:5000` | 节点侧镜像仓库地址 |
| `global.image.repository` | `how-dev-iam` | 镜像组织路径 |
| `global.image.tag` | `local` | 镜像 tag（各子 chart 可覆盖） |
| `global.image.pullPolicy` | `IfNotPresent` | 拉取策略 |
| `global.config.mysql.host` | `middleware-mysql.iam-middleware.svc.cluster.local` | MySQL FQDN |
| `global.config.mysql.port/user/password/database` | 见 `values.yaml` | MySQL 连接参数 |
| `iam-account.enabled` | `true` | 开关 iam-account 子 chart |
| `iam-front.enabled` | `true` | 开关 iam-front 子 chart |

### `iam-account`

| Key | 默认 | 说明 |
|---|---|---|
| `replicaCount` | `1` | 副本数 |
| `service.httpPort` | `8001` | 与 `apps/backend/iam-account/conf/config.json` 保持一致 |
| `service.grpcPort` | `0` | 0 表示不启用 gRPC |
| `service.metricsPort` | `9091` | Prometheus 指标 |
| `database` | `iam_account` | 该服务使用的数据库名 |
| `config.*` | 见 `values.yaml` | 渲染进 `config.json` |

生成的 `config.json` 结构必须与 `apps/backend/iam-account/logic/config/config.go` 中的 `Detail` 结构体完全对齐。

### `iam-front`

| Key | 默认 | 说明 |
|---|---|---|
| `service.port` | `80` | Service port |
| `backendService` | `iam-account` | nginx 反代目标 Service |
| `backendPort` | `8001` | nginx 反代目标端口 |

nginx `default.conf` 由 configmap 挂载生成，包含：
- SPA fallback：`try_files $uri $uri/ /index.html`
- 后端反代：`location /api/ → proxy_pass http://<backendService>:<backendPort>` （**不带尾斜杠**，保留 `/api/` 前缀原样透传给后端 gin 的 `/api` 路由组）
- 探活端点：`GET /healthz → 200 ok`

### `middleware`

| Key | 默认 | 说明 |
|---|---|---|
| `mysql.rootPassword` | `IamR00tPwd#2026` | MySQL root 密码 |
| `mysql.database` | `iam_account` | 默认创建的业务库 |
| `mysql.extraDatabases` | `[iam_authn, iam_authz]` | 额外创建的库 |
| `mysql.persistence.enabled` | `false` | 是否使用 PVC；本地开发默认 emptyDir |
| `redis.password` | `IamR3d1sPwd#2026` | Redis 密码 |
| `redis.persistence.enabled` | `false` | 是否持久化 |

### `init-jobs`

| Key | 默认 | 说明 |
|---|---|---|
| `mysqlSchema.enabled` | `true` | 是否执行 schema 初始化 |
| `mysqlSchema.mysql.host` | `middleware-mysql` | 目标 MySQL host；跨 namespace 请传 FQDN |
| `mysqlSchema.services[]` | `[iam-account]` | 需要初始化的服务列表；对应 `ddl/<name>/*.sql` |

## 常用 helm 命令

```bash
# 校验
helm lint deploy/k8s/helm/k8s-infra
helm lint deploy/k8s/helm/middleware
helm lint deploy/k8s/helm/init-jobs
helm lint deploy/k8s/helm/iam

# 拉依赖（k8s-infra 需要）
helm dependency update deploy/k8s/helm/k8s-infra

# 渲染 & 查看最终 yaml
helm template iam deploy/k8s/helm/iam \
  --set global.image.registry=host.docker.internal:5000 \
  --set global.image.tag=local

# 手动部署（部署脚本内部等价于以下四条）
helm upgrade -i k8s-infra  deploy/k8s/helm/k8s-infra  -n ingress-nginx --create-namespace --wait \
  --set iamIngress.namespace=iam --set iamIngress.host=iam.local
helm upgrade -i middleware deploy/k8s/helm/middleware -n iam-middleware --create-namespace --wait
helm upgrade -i iam-init   deploy/k8s/helm/init-jobs  -n iam --create-namespace --wait \
  --set mysqlSchema.mysql.host=middleware-mysql.iam-middleware.svc.cluster.local
helm upgrade -i iam        deploy/k8s/helm/iam        -n iam --wait \
  --set global.image.registry=host.docker.internal:5000 \
  --set global.image.repository=how-dev-iam \
  --set global.image.tag=local

# 卸载
helm uninstall iam       -n iam
helm uninstall iam-init  -n iam
helm uninstall middleware -n iam-middleware
helm uninstall k8s-infra -n ingress-nginx
```

## 与源码字段的对应关系

- iam-account [`config.go`](../../../apps/backend/iam-account/logic/config/config.go) 中的 `Detail` 决定了 configmap 生成的 `config.json` 结构。
- iam-front [`vite.config.ts`](../../../apps/frontend/iam-front/vite.config.ts) 中的 `/api → :8001` dev proxy 与本 chart 中 `iam-front` nginx 的 `/api` 反代等价。
- iam-account [`router.go`](../../../apps/backend/iam-account/logic/route/router.go) 中的 `/healthCheck/status` 是 deployment probe 的健康检查路径。
