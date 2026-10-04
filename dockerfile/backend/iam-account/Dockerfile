# =============================================================================
# iam-account Dockerfile (multi-stage)
# -----------------------------------------------------------------------------
# 用法（在项目根目录 how-dev-iam/ 下执行）：
#   docker build \
#     -t host.docker.internal:5000/how-dev-iam/iam-account:local \
#     -f dockerfile/backend/iam-account/Dockerfile .
#
# 说明：
#   1) 构建上下文期望为 how-dev-iam/ 项目根目录，构建时进入
#      apps/backend/iam-account 目录执行 go build。
#   2) 运行阶段监听 HTTP=8001 / metrics=9091 / pprof=6061，
#      与 helm chart / config.json 保持一致。
# =============================================================================

# ---------- Build stage ----------
FROM golang:1.23-alpine AS builder

# Alpine 里跑 CGO 需要 gcc；此处禁用 CGO，纯静态编译。
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GO111MODULE=on \
    GOPROXY=https://proxy.golang.org,direct

WORKDIR /workspace

# 只把 iam-account 服务源码拷进来（构建上下文为项目根，这里显式指定路径）
COPY apps/backend/iam-account/ ./iam-account/

WORKDIR /workspace/iam-account

# 依赖下载 + 构建
RUN go mod download && \
    go build -trimpath -ldflags="-s -w" -o /out/iam-account .

# ---------- Runtime stage ----------
FROM alpine:3.20

# ca-certificates: 服务未来调外部 HTTPS 时需要；tzdata：让日志时间正确
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -H -u 10001 iam

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /out/iam-account /app/iam-account

# helm chart 会把 config.json 挂到 /config/config.json；这里只声明预期路径
# （通过 args 传递，见 deployment.yaml）
USER iam

EXPOSE 8001 9091 6061

ENTRYPOINT ["/app/iam-account"]
CMD ["-config", "/config/config.json"]
