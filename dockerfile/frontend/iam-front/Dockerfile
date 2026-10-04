# =============================================================================
# iam-front Dockerfile (multi-stage)
# -----------------------------------------------------------------------------
# 用法（在项目根目录 how-dev-iam/ 下执行）：
#   docker build \
#     -t host.docker.internal:5000/how-dev-iam/iam-front:local \
#     -f dockerfile/frontend/iam-front/Dockerfile .
#
# 说明：
#   1) 构建阶段用 node:20-alpine 编译 vite 产物到 dist/。
#   2) 运行阶段用 nginx:1.27-alpine 提供静态文件；default.conf 由 helm
#      configmap 在部署时挂载覆盖（承担 SPA fallback + /api 反代）。
# =============================================================================

# ---------- Build stage ----------
FROM node:20-alpine AS builder

WORKDIR /workspace

# 只拷贝前端源码；构建上下文为项目根，这里显式指定路径
COPY apps/frontend/iam-front/package.json apps/frontend/iam-front/package-lock.json* ./
RUN npm ci --no-audit --no-fund

# 拷源代码（在依赖装完后再拷，充分利用 Docker layer 缓存）
COPY apps/frontend/iam-front/ ./

# 构建产物：/workspace/dist
RUN npm run build

# ---------- Runtime stage ----------
FROM nginx:1.27-alpine

# 兜底 default.conf；实际部署时会被 helm ConfigMap 挂载覆盖
RUN rm -f /etc/nginx/conf.d/default.conf
COPY --from=builder /workspace/dist /usr/share/nginx/html

EXPOSE 80

# 使用 nginx 默认 CMD 即可
