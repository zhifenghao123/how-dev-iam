# how-dev-iam

一个面向开发者场景的 IAM（Identity and Access Management，身份与访问管理）平台。

## 项目结构

```
apps/
└── backend/
    ├── iam-account/   # 账号
    ├── iam-authn/     # 认证 (Authentication)
    ├── iam-authz/     # 授权 (Authorization)
    └── iam-authv/     # 鉴权 (Access Verification)
```

## 后端模块介绍（`apps/backend`）

后端由四个相互协作、职责单一的微服务组成，共同覆盖 IAM 的完整能力闭环：**账号 → 认证 → 授权 → 鉴权**。

### 1. iam-account（账号）

- **路径**：`apps/backend/iam-account`
- **职责**：负责账号（Identity）全生命周期的管理，是整个 IAM 系统的数据基石。
- **核心能力**：
    - 用户、组织、部门、用户组等主体的增删改查
    - 账号注册、注销、启用/禁用、密码管理与重置
    - 账号属性、身份来源（本地 / LDAP / 外部 IdP）管理
    - 为 `iam-authn`、`iam-authz` 提供统一的主体数据来源

### 2. iam-authn（认证）

- **路径**：`apps/backend/iam-authn`
- **职责**：解决"你是谁"（Authentication）的问题，负责校验用户身份并颁发凭证。
- **核心能力**：
    - 多种登录方式：用户名/密码、短信/邮箱验证码、扫码、MFA 多因素认证
    - 单点登录协议支持：OIDC / OAuth2.0 / SAML 2.0 / CAS
    - 登录态与会话管理（Session / Token / Refresh Token）
    - 登录风控、失败锁定、登录审计

### 3. iam-authz（授权）

- **路径**：`apps/backend/iam-authz`
- **职责**：解决"你能做什么"（Authorization）的问题，负责权限策略的定义、分配与管理。
- **核心能力**：
    - 权限模型：RBAC（角色）/ ABAC（属性）/ ReBAC（关系）
    - 角色、权限点、资源、策略（Policy）的建模与管理
    - 授权关系维护：给谁（Subject）在什么资源（Resource）上赋予什么操作（Action）
    - 授权变更审计与策略版本管理

### 4. iam-authv（鉴权）

- **路径**：`apps/backend/iam-authv`
- **职责**：解决"当前这次请求是否被允许"（Access Verification / Enforcement）的问题，是权限的**运行时判定与拦截**入口。
- **核心能力**：
    - 基于 `iam-authn` 颁发的凭证解析身份
    - 基于 `iam-authz` 定义的策略执行 PDP（Policy Decision Point）判定
    - 高性能鉴权接口（Check / BatchCheck）
    - 面向网关、业务服务的 SDK / 中间件接入
    - 鉴权结果缓存与访问审计日志

## 模块协作关系

```
                ┌──────────────┐
                │  iam-account │  账号 & 组织数据
                └──────┬───────┘
                       │ 提供主体
        ┌──────────────┼──────────────┐
        ▼                             ▼
┌──────────────┐               ┌──────────────┐
│   iam-authn  │  身份认证     │   iam-authz  │  策略管理
│  (你是谁)    │──────凭证─────▶│  (你能做什么)│
└──────┬───────┘               └──────┬───────┘
       │                              │
       └───────────┬──────────────────┘
                   ▼
            ┌──────────────┐
            │   iam-authv  │  运行时鉴权
            │  (是否允许)  │
            └──────────────┘
```

> 简单记忆：**account 建人 → authn 验人 → authz 配权 → authv 判权**。