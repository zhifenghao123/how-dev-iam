# how-dev-iam 部署指南

把 how-dev-iam 平台（`iam-account` 后端 + `iam-front` 前端 + 依赖的中间件）部署到本机 Colima k3s 集群，镜像走本地私有 Docker Registry（Basic Auth）。

- 部署脚本目录：[`scripts/`](scripts/)
- Helm chart 目录：[`k8s/helm/`](k8s/helm/)
- 镜像 Dockerfile：[`../dockerfile/`](../dockerfile/)

---

## 目录

- [1. 拓扑](#1-拓扑)
- [2. 脚本一览](#2-脚本一览)
- [3. 快速开始](#3-快速开始)
- [4. 各脚本详解](#4-各脚本详解)
- [5. 环境变量总表](#5-环境变量总表)
- [6. 日常运维](#6-日常运维)
- [7. 故障排查（FAQ）](#7-故障排查faq)

---

## 1. 拓扑

```
┌─────────────────────────── Mac 宿主机 ────────────────────────────┐
│  Local Docker Registry (localhost:5000, http)                      │
│    ↑ docker push (build.sh)                                        │
│                                                                    │
│  ┌────────────────── Colima VM (k3s node) ─────────────────────┐   │
│  │  docker daemon (insecure-registry: host.docker.internal:5000)│  │
│  │    ↑ pull ← host.docker.internal:5000                        │  │
│  │  ┌──────────── k3s (cri-dockerd) ─────────────────────────┐  │  │
│  │  │  ns: ingress-nginx    ← k8s-infra chart                │  │  │
│  │  │    └── ingress-nginx-controller (LoadBalancer, :80)    │  │  │
│  │  │  ns: iam-middleware   ← middleware chart               │  │  │
│  │  │    ├── middleware-mysql   (Service:3306)               │  │  │
│  │  │    └── middleware-redis   (Service:6379)               │  │  │
│  │  │  ns: iam              ← init-jobs + iam charts         │  │  │
│  │  │    ├── (Job) init-mysql-schema-iam-account             │  │  │
│  │  │    ├── iam-account (Service:8001, metrics:9091)        │  │  │
│  │  │    ├── iam-front   (Service:80, nginx SPA+/api proxy)  │  │  │
│  │  │    └── Ingress "iam"  host=iam.local → iam-front       │  │  │
│  │  │        （由 k8s-infra chart 部署）                     │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────┘ │
│  浏览器 → http://iam.local (hosts: 127.0.0.1 iam.local)            │
└────────────────────────────────────────────────────────────────────┘
```

**Helm release 分布**：

| Release | Namespace | Chart 路径 | 内容 |
|---|---|---|---|
| `k8s-infra` | `ingress-nginx` | `k8s/helm/k8s-infra` | ingress-nginx controller + iam Ingress 资源 |
| `middleware` | `iam-middleware` | `k8s/helm/middleware` | mysql + redis |
| `iam-init` | `iam` | `k8s/helm/init-jobs` | mysql schema 初始化 Job |
| `iam` | `iam` | `k8s/helm/iam` | iam-account + iam-front |

---

## 2. 脚本一览

所有脚本都在 [`deploy/scripts/`](scripts/) 下，都可以**独立执行**，也都**自带最小前置检查**（fail-fast 时会打印可复制的修复命令）。

| 脚本 | 职责 | 是否修改集群 |
|---|---|---|
| [`preflight.sh`](scripts/preflight.sh) | **环境体检**（10 步 / 24 项），只报告，不修改 | ❌ |
| [`build.sh`](scripts/build.sh) | 构建 & push 后端 / 前端镜像 | ❌ |
| [`deploy-infra.sh`](scripts/deploy-infra.sh) | 部署 k8s-infra（ingress-nginx + iam Ingress） | ✅ |
| [`deploy-middleware.sh`](scripts/deploy-middleware.sh) | 部署 mysql / redis | ✅ |
| [`deploy-init.sh`](scripts/deploy-init.sh) | 执行数据库 schema 初始化 Job | ✅ |
| [`deploy-iam.sh`](scripts/deploy-iam.sh) | 部署 iam 业务（iam-account + iam-front） | ✅ |
| [`deploy-all.sh`](scripts/deploy-all.sh) | 一键顺序编排 infra/middleware/init/iam **四个部署脚本**（不含 build） | ✅ |
| [`deploy-headlamp.sh`](scripts/deploy-headlamp.sh) | 【可选】部署 Headlamp（k8s Web Dashboard），独立 release，**不进入 deploy-all.sh** | ✅ |

**核心原则**：
- **preflight.sh**：只**检测**，不**修改**。发现缺失项时打印可复制的修复命令。
- **每个 deploy-*.sh**：只做自己那一步；开始时校验自己那一步所需的最小前置条件（脚本内部检查，无需先跑 preflight，但推荐首次部署前跑一次完整体检）。
- **deploy-all.sh**：仅编排（按序调用其他脚本），不做具体动作。

---

## 3. 快速开始

### 3.1 首次部署

```bash
cd /Users/zefinnhao/SoftwareDevelop/projects/hzf_project/how-dev-iam

# ① 完整体检；按输出的 [ERR] 修复建议一一处理
bash deploy/scripts/preflight.sh

# ② 构建 & push 镜像
bash deploy/scripts/build.sh

# ③ 一键部署（infra → middleware → init → iam）
bash deploy/scripts/deploy-all.sh

# ④ 浏览器打开
open http://iam.local
```

### 3.2 分步部署（推荐用于调试）

```bash
bash deploy/scripts/build.sh              # 构建 & push
bash deploy/scripts/deploy-infra.sh       # 基础公共组件（ingress-nginx + Ingress）
bash deploy/scripts/deploy-middleware.sh  # mysql / redis
bash deploy/scripts/deploy-init.sh        # 建表
bash deploy/scripts/deploy-iam.sh         # 业务
```

### 3.3 增量场景（按需）

```bash
# 只改了 iam-account 代码 → 重建业务
bash deploy/scripts/build.sh
bash deploy/scripts/deploy-iam.sh

# 只改了 chart → 只重新 helm upgrade
bash deploy/scripts/deploy-iam.sh --skip-image-check

# 只想构建镜像不部署
bash deploy/scripts/build.sh

# 只构建前端
bash deploy/scripts/build.sh --only front
```

---

## 4. 各脚本详解

### 4.1 `preflight.sh` — 完整环境体检

**行为**：只检测、只报告，**永不改动**你的系统。缺失项打印可复制的修复命令。

**检查项（10 步 / 共 24 个子项）**：

| # | 检查内容 |
|---|---|
| 1 | CLI 工具是否安装：docker / colima / kubectl / helm / make |
| 2 | docker daemon 是否可用 |
| 3 | Colima profile `k8s` 是否存在且 Running |
| 4 | kubectl 能否访问集群、node 是否 Ready |
| 5 | 本地私有 registry `localhost:5000` 端口是否可达 + `/v2/` 是否响应；若开启 Basic Auth 则校验凭据；宿主机 docker 登录状态 |
| 6 | Colima VM 内 `/etc/docker/daemon.json` 是否含 insecure-registry |
| 7 | ingress-nginx 是否安装并 Ready、IngressClass 是否存在 |
| 8 | `/etc/hosts` 是否含 `iam.local` 映射 |
| 9 | 宿主机 80/443 端口占用情况 |
| 10 | Chart / Dockerfile 是否齐全、`helm lint` 是否通过 |

**退出码**：0 = 全部通过；1 = 存在阻塞项

### 4.2 `build.sh` — 构建镜像

**内置最小检查**：docker daemon、本地 registry 可达、Basic Auth 凭据有效。

**行为**：push 前会用 `--password-stdin` 幂等执行 `docker login`（不进 shell history）。

```bash
bash deploy/scripts/build.sh                # 构建两个
bash deploy/scripts/build.sh --only account # 只构建后端
bash deploy/scripts/build.sh --only front   # 只构建前端
bash deploy/scripts/build.sh --no-push      # 只 build 不 push（也不 login）
bash deploy/scripts/build.sh --dry-run
```

### 4.3 `deploy-infra.sh` — 基础公共组件

**部署内容**：
- 官方 `ingress-nginx` chart（作为 dependency 引入，装在 `ingress-nginx` ns）
- iam Ingress 资源（`host=iam.local` → `iam-front`，装在 `iam` ns）

**内置最小检查**：helm、kubectl 集群可访问。

```bash
bash deploy/scripts/deploy-infra.sh
bash deploy/scripts/deploy-infra.sh --skip-dep-update  # 已下载过 ingress-nginx tgz
bash deploy/scripts/deploy-infra.sh --dry-run
```

### 4.4 `deploy-middleware.sh` — 中间件

**部署内容**：mysql 8.0 + redis 7，都在 `iam-middleware` ns。

**内置最小检查**：helm、kubectl 集群可访问。

```bash
bash deploy/scripts/deploy-middleware.sh
```

### 4.5 `deploy-init.sh` — 数据库 Schema

**部署内容**：`iam-init` helm release，通过 helm hook 触发一次性 Job 执行 `ddl/iam-account/*.sql`。

**内置最小检查**：helm、kubectl 集群可访问、**middleware-mysql Service 已就绪**（endpoint 非空）。

```bash
bash deploy/scripts/deploy-init.sh
```

### 4.6 `deploy-iam.sh` — 业务

**部署内容**：iam-account + iam-front。

**内置最小检查**：
- helm、kubectl 集群可访问
- middleware-mysql Service 就绪
- ingress-nginx-controller Deployment Ready
- **两个镜像在本地 registry 中确实存在**（v2 API HEAD，带 Basic Auth）

**自动动作**：在 `iam` ns 幂等创建/更新 `Secret/image-credentials`（`kubernetes.io/dockerconfigjson`），Chart 通过 `global.imagePullSecrets` 引用。

```bash
bash deploy/scripts/deploy-iam.sh
bash deploy/scripts/deploy-iam.sh --skip-image-check   # 跳过镜像存在性检查
bash deploy/scripts/deploy-iam.sh --dry-run
```

### 4.7 `deploy-all.sh` — 部署编排

**行为**：按 `infra → middleware → init → iam` 顺序调用其他 4 个部署脚本；自己**不做**具体动作。

**注意**：该脚本**只负责部署**，不涉及镜像构建。如需构建，请先单独执行 `bash deploy/scripts/build.sh`。

```bash
bash deploy/scripts/deploy-all.sh                              # 完整部署流程
bash deploy/scripts/deploy-all.sh --from middleware            # 从 middleware 开始
bash deploy/scripts/deploy-all.sh --skip init                  # 跳过某步（可多次）
bash deploy/scripts/deploy-all.sh --skip middleware --skip init
bash deploy/scripts/deploy-all.sh --dry-run
```

### 4.8 `deploy-headlamp.sh` — 【可选】Kubernetes Web Dashboard

**用途**：部署 [Headlamp](https://headlamp.dev/)（k8s Dashboard 的开源实现），提供 Web UI 查看/管理集群资源，方便本地调试。

**独立性**：单独的 helm release + namespace（默认 `headlamp`），**不进入 `deploy-all.sh` 编排流程**；卸载不影响 iam 平台。

**内置最小检查**：helm、kubectl 集群可访问；启用 `--with-ingress` 时会校验 ingress-nginx 就绪。

**做的事**：
1. `helm repo add headlamp https://kubernetes-sigs.github.io/headlamp/`
2. `helm upgrade -i headlamp headlamp/headlamp -n headlamp`（ClusterIP Service）
3. 创建 `ServiceAccount/headlamp-admin` + `ClusterRoleBinding/headlamp-admin-cluster-admin`
4. 通过 `kubectl create token --duration=24h` 生成登录 token 并打印

```bash
# 基本部署（默认走 port-forward 访问）
bash deploy/scripts/deploy-headlamp.sh

# 部署 + 前台起 port-forward，Ctrl+C 停止
bash deploy/scripts/deploy-headlamp.sh --port-forward

# 部署 + Ingress（host=headlamp.local，需 /etc/hosts 有映射）
bash deploy/scripts/deploy-headlamp.sh --with-ingress

# 重新生成登录 token（token 24h 有效期）
bash deploy/scripts/deploy-headlamp.sh --print-token

# 一键卸载（release / SA / ClusterRoleBinding / namespace 全清理）
bash deploy/scripts/deploy-headlamp.sh --uninstall
```

**访问步骤**：
1. 执行 `bash deploy/scripts/deploy-headlamp.sh`
2. 复制脚本末尾输出的 Bearer Token
3. 在另一个终端起转发：`kubectl -n headlamp port-forward svc/headlamp 8090:80`
4. 浏览器打开 <http://localhost:8090>，登录页选 "Token" 方式粘贴上一步的 token

**环境变量**（可选）：`HEADLAMP_NS` / `HEADLAMP_RELEASE` / `HEADLAMP_SA` / `HEADLAMP_ROLE` / `HEADLAMP_HOST` / `HEADLAMP_PORT`。

---

## 5. 环境变量总表

所有脚本共用一组环境变量，未设时使用下表默认值：

| 变量 | 默认 | 说明 |
|---|---|---|
| `REGISTRY_HOST` | `localhost:5000` | 宿主机 push 地址 |
| `REGISTRY_NODE` | `host.docker.internal:5000` | k8s 节点 pull 地址 |
| `REGISTRY_ORG` | `how-dev-iam` | 镜像组织路径 |
| `IMAGE_TAG` | `local` | 镜像 tag |
| `REGISTRY_USER` | `admin` | 私有 registry 用户名（Basic Auth） |
| `REGISTRY_PASSWORD` | `admin123` | 私有 registry 密码（Basic Auth） |
| `IMAGE_PULL_SECRET` | `image-credentials` | pod 拉镜像用的 Secret 名（部署脚本自动创建） |
| `IAM_NS` | `iam` | 业务命名空间 |
| `MW_NS` | `iam-middleware` | 中间件命名空间 |
| `INFRA_NS` | `ingress-nginx` | 基础公共组件命名空间 |
| `IAM_HOST` | `iam.local` | Ingress host |
| `COLIMA_PROFILE` | `k8s` | Colima profile 名（preflight 使用） |

**关于凭据的默认值**：与 `how-dev-cloud-native` 项目 `install.sh` 的出厂凭据一致（`admin/admin123`）。如自定义过，请显式导出：

```bash
export REGISTRY_USER=myuser
export REGISTRY_PASSWORD='my-strong-pwd'
bash deploy/scripts/build.sh
bash deploy/scripts/deploy-iam.sh
```

示例（用 git commit 短 sha 作 tag）：

```bash
IMAGE_TAG=$(git rev-parse --short HEAD) bash deploy/scripts/build.sh && bash deploy/scripts/deploy-all.sh
```

---

## 6. 日常运维

| 场景 | 命令 |
|---|---|
| 改了后端代码 | `bash deploy/scripts/build.sh --only account && bash deploy/scripts/deploy-iam.sh` |
| 改了前端代码 | `bash deploy/scripts/build.sh --only front && bash deploy/scripts/deploy-iam.sh` |
| 改了 iam chart | `bash deploy/scripts/deploy-iam.sh --skip-image-check` |
| 卸载业务 | `helm uninstall iam iam-init -n iam` |
| 卸载中间件 | `helm uninstall middleware -n iam-middleware` |
| 卸载基础设施 | `helm uninstall k8s-infra -n ingress-nginx` |
| 全部卸载 | 依次 uninstall 4 个 release |
| iam-account 日志 | `kubectl -n iam logs -l app.kubernetes.io/name=iam-account --tail=200 -f` |
| iam-front 日志 | `kubectl -n iam logs -l app.kubernetes.io/name=iam-front --tail=200 -f` |
| MySQL 日志 | `kubectl -n iam-middleware logs -l app.kubernetes.io/name=mysql --tail=200 -f` |
| MySQL 端口转发 | `kubectl -n iam-middleware port-forward svc/middleware-mysql 3306:3306` |
| 后端端口转发 | `kubectl -n iam port-forward svc/iam-account 8001:8001` |
| Ingress 兜底访问 | `kubectl -n ingress-nginx port-forward svc/ingress-nginx-controller 8080:80` |

---

## 7. 故障排查（FAQ）

### Q1：pod ImagePullBackOff，事件写 `http: server gave HTTP response to HTTPS client`

**原因**：Colima VM 内 docker daemon 未把 `host.docker.internal:5000` 加入 insecure-registries。

**修复**：

```bash
colima ssh -p k8s -- sudo sh -c 'python3 -c "import json,os; p=\"/etc/docker/daemon.json\"; d=json.load(open(p)) if os.path.exists(p) else {}; d[\"insecure-registries\"]=list(set(d.get(\"insecure-registries\",[])+[\"host.docker.internal:5000\",\"localhost:5000\"])); open(p,\"w\").write(json.dumps(d,indent=2))" && systemctl restart docker'
```

### Q2：pod 拉镜像时 `dial tcp: lookup host.docker.internal: no such host`

**原因**：Colima VM 内 `host.docker.internal` 未解析到宿主机。

**修复**：

```bash
# 查 VM 视角的宿主机 IP
colima ssh -p k8s -- ip route show default | awk '{print $3}'
# 用该 IP 作为 REGISTRY_NODE 重新部署
REGISTRY_NODE=192.168.5.2:5000 bash deploy/scripts/deploy-iam.sh --skip-image-check
```

### Q3：iam-account CrashLoopBackOff，日志 `mysql.dsn is required`

**原因**：`configmap` 生成时 `.Values.global.config.mysql` 被清空。

**修复**：检查 `deploy/k8s/helm/iam/values.yaml` 中的 `global.config.mysql` 是否完整。

### Q4：iam-account 连不上 MySQL，日志 `dial tcp ... i/o timeout`

**原因**：`middleware-mysql` 未 Ready 或 FQDN 拼错。

**修复**：

```bash
kubectl -n iam-middleware get pods -l app.kubernetes.io/name=mysql
kubectl -n iam-middleware logs -l app.kubernetes.io/name=mysql --tail=100
```

### Q5：`kubectl get ing` ADDRESS 为空

**原因**：ingress-nginx 未装或未就绪。

**修复**：

```bash
bash deploy/scripts/deploy-infra.sh
kubectl -n ingress-nginx wait --for=condition=Available deploy/ingress-nginx-controller --timeout=180s
```

### Q6：浏览器访问 `http://iam.local` 转圈

**排查**：

```bash
grep iam.local /etc/hosts                     # 1) hosts 映射
sudo lsof -i :80 -sTCP:LISTEN                 # 2) 80 端口占用
kubectl -n ingress-nginx get svc              # 3) EXTERNAL-IP
# 兜底方案
kubectl -n ingress-nginx port-forward svc/ingress-nginx-controller 8080:80
# 访问 http://iam.local:8080
```

### Q7：切到 NodePort 方案暴露 ingress

如果 Colima 端口 forward 有问题（80 无法访问），可以直接把 ingress-nginx service 改成 NodePort：

```bash
helm upgrade -i k8s-infra deploy/k8s/helm/k8s-infra -n ingress-nginx \
  --set ingress-nginx.controller.service.type=NodePort \
  --set ingress-nginx.controller.service.nodePorts.http=30080
# 访问 http://iam.local:30080
```

### Q8：宿主机 `docker push localhost:5000/...` 报 http/https 错

**修复**：宿主机 Docker daemon (Docker Desktop / colima default profile) 也要把 `localhost:5000` 加入 insecure-registries。

### Q9：pod ImagePullBackOff，事件写 `no basic auth credentials` 或 `unauthorized`

**原因**：本地 registry 开启了 Basic Auth，但 pod 拉镜像时没带凭据。

**排查**：

```bash
# 1) 确认 Secret 存在
kubectl -n iam get secret image-credentials -o yaml | head -30

# 2) 确认 Deployment 引用了它
kubectl -n iam get deploy iam-account -o jsonpath='{.spec.template.spec.imagePullSecrets}'
# 期望输出：[{"name":"image-credentials"}]

# 3) 手动重建 Secret（等价于 deploy-iam.sh 做的事）
kubectl -n iam create secret docker-registry image-credentials \
  --docker-server=host.docker.internal:5000 \
  --docker-username=admin \
  --docker-password=admin123 \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Q10：宿主机 `docker login localhost:5000` 报错

**排查**：

```bash
# 1) registry 是否在线
curl -sI http://localhost:5000/v2/    # 期望 401 Unauthorized（说明认证在生效）

# 2) 凭据是否正确
curl -sI -u admin:admin123 http://localhost:5000/v2/   # 期望 200

# 3) 手动登录
echo 'admin123' | docker login localhost:5000 -u admin --password-stdin
```
