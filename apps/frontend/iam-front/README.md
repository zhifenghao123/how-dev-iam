# iam-front

how-dev-iam 平台前端（React 18 + TypeScript + AntD 5 + Vite）。

## 目录结构

```
src/
├── api/           axios 封装 + 后端接口调用
├── layouts/       通用 Layout
├── pages/         页面
├── stores/        Zustand 状态
├── App.tsx
├── main.tsx
└── router.tsx
```

## 本地开发

```bash
# 前置：确保 iam-account 后端已启动在 :8001（默认）
cd apps/frontend/iam-front
npm install
npm run dev
```

访问 http://localhost:5173

## 页面 & 闭环

- `/register` 个人注册（邮箱 + 验证码 + 密码）
- `/login` 登录（临时调试入口；正式登录由 iam-authn 接入后替换）
- `/dashboard` 我的账户列表 + 切换工作空间
- `/register-org` 创建企业账户
- `/accounts/:aid/members` 成员管理（邀请 / 直接创建子用户 / 移除）
- `/invitations/:token` 接受邀请

## 与后端的约定

- axios 默认 `baseURL = /api`，Vite dev proxy 到 `127.0.0.1:8001`
- 请求头 `X-User-Id` 携带当前登录 user_id（临时；未来接 iam-authn 后改为 Bearer token）
- 响应格式：`{ code: 'OK', data: ... }`，错误 `{ code, message }`
