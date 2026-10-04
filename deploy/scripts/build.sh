#!/usr/bin/env bash
# =============================================================================
# deploy/scripts/build.sh
# -----------------------------------------------------------------------------
# 用途：构建 how-dev-iam 后端 / 前端镜像，并 push 到本地私有 Registry。
#
# 内置最小前置检查：
#   - docker CLI + daemon 可用
#   - 本地 registry ${REGISTRY_HOST} 在线、凭据合法（开了 Basic Auth）
#   - push 前自动 docker login（幂等）
#
# 使用：
#   bash deploy/scripts/build.sh                # 构建两个镜像
#   bash deploy/scripts/build.sh --only account # 只构建 iam-account
#   bash deploy/scripts/build.sh --only front   # 只构建 iam-front
#   bash deploy/scripts/build.sh --no-push      # 只 build 不 push（也不 login）
#   bash deploy/scripts/build.sh --dry-run      # 只打印命令
#
# 环境变量：
#   REGISTRY_HOST     (默认 localhost:5000)
#   REGISTRY_ORG      (默认 how-dev-iam)
#   IMAGE_TAG         镜像 tag。未指定时按以下规则派生（优先级从高到低）：
#                       1) IMAGE_TAG=xxx           → 直接使用
#                       2) TAG=xxx                 → 作为 IMAGE_TAG 别名
#                       3) VERSION=vX.Y.Z          → "${VERSION}-<yyyymmdd-HHMMSS>"
#                       4) 都不指定                 → "v0.0.1-<yyyymmdd-HHMMSS>"
#                     生成后会写入 deploy/.deploy/last-image-tag，供 deploy-* 复用。
#   VERSION           基线版本号（默认 v0.0.1），仅在派生 IMAGE_TAG 时使用
#   TAG               IMAGE_TAG 的别名
#   REGISTRY_USER     (默认 admin)      — 本地 registry 基本认证用户名
#   REGISTRY_PASSWORD (默认 admin123)   — 本地 registry 基本认证密码
# =============================================================================

# ---------- 强制 bash 执行（防止用 `sh xxx.sh` 调用导致行为异常）----------
# macOS 下 `sh` 实际是 POSIX 限制模式，`echo -e` / `[[ ]]` / 数组 / `set -o pipefail`
# 等 bashism 均不支持。若发现不是 bash，自动 re-exec。
# 注意：sh 启动时会设置 POSIXLY_CORRECT=y，会传给新 bash 让它进入 POSIX 模式，
# 导致 `echo -e` 依然把 `-e` 原样输出。必须在 re-exec 前 unset 掉。
if [ -z "${BASH_VERSION:-}" ]; then
    unset POSIXLY_CORRECT
    exec /usr/bin/env bash "$0" "$@"
fi
# 如果是 bash 但被强制进入了 POSIX 模式（例如 sh -> bash 链），也切回来
if [ -n "${POSIXLY_CORRECT:-}" ]; then
    unset POSIXLY_CORRECT
    set +o posix 2>/dev/null || true
fi

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
PROJECT_ROOT="$(resolve_project_root "${SCRIPT_DIR}/lib")"

REGISTRY_HOST="${REGISTRY_HOST:-localhost:5000}"
REGISTRY_ORG="${REGISTRY_ORG:-how-dev-iam}"
REGISTRY_USER="${REGISTRY_USER:-admin}"
REGISTRY_PASSWORD="${REGISTRY_PASSWORD:-admin123}"
# IMAGE_TAG 按 IMAGE_TAG > TAG > "${VERSION}-<时间戳>" > "v0.0.1-<时间戳>" 顺序解析
IMAGE_TAG="$(resolve_image_tag "${PROJECT_ROOT}")"

ACCOUNT_IMAGE="${REGISTRY_HOST}/${REGISTRY_ORG}/iam-account:${IMAGE_TAG}"
FRONT_IMAGE="${REGISTRY_HOST}/${REGISTRY_ORG}/iam-front:${IMAGE_TAG}"

# ---------- 参数 ----------
ONLY=""
DO_PUSH=1
DRY_RUN=0

usage() {
    cat <<EOF
用法：$(basename "$0") [选项]

选项：
  --only <name>   只构建 account 或 front
  --no-push       构建后不 push
  --dry-run       只打印命令
  -h, --help      显示帮助
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --only)     ONLY="${2:-}"; shift ;;
        --no-push)  DO_PUSH=0 ;;
        --dry-run)  DRY_RUN=1 ;;
        -h|--help)  usage; exit 0 ;;
        *) log_error "未知参数：$1"; usage; exit 2 ;;
    esac
    shift
done
export DRY_RUN

if [[ -n "${ONLY}" && "${ONLY}" != "account" && "${ONLY}" != "front" ]]; then
    log_error "--only 仅支持 account|front，收到：${ONLY}"; exit 2
fi

# ---------- 打印计划 ----------
print_env_summary "构建镜像" \
    "项目根路径      : ${PROJECT_ROOT}" \
    "Registry (push) : ${REGISTRY_HOST}" \
    "Registry 用户   : ${REGISTRY_USER}" \
    "Image tag       : ${IMAGE_TAG}" \
    "目标镜像        : ${ACCOUNT_IMAGE}" \
    "               : ${FRONT_IMAGE}" \
    "构建目标        : ${ONLY:-全部}" \
    "是否 push       : $([[ ${DO_PUSH} == 1 ]] && echo yes || echo no)"

# ---------- 前置检查 ----------
log_step 1 4 "前置检查"
require_docker_daemon
if [[ "${DO_PUSH}" == "1" ]]; then
    require_registry "${REGISTRY_HOST}" "${REGISTRY_USER}" "${REGISTRY_PASSWORD}"
fi
log_ok "前置检查通过"

# ---------- docker login（push 前幂等登录）----------
if [[ "${DO_PUSH}" == "1" ]]; then
    log_step 2 4 "docker login ${REGISTRY_HOST}"
    docker_login_registry "${REGISTRY_HOST}" "${REGISTRY_USER}" "${REGISTRY_PASSWORD}"
else
    log_step 2 4 "docker login（已跳过：--no-push）"
fi

# ---------- 构建 iam-account ----------
if [[ -z "${ONLY}" || "${ONLY}" == "account" ]]; then
    log_step 3 4 "构建 iam-account 镜像"
    run_cmd docker build \
        -t "${ACCOUNT_IMAGE}" \
        -f "${PROJECT_ROOT}/dockerfile/backend/iam-account/Dockerfile" \
        "${PROJECT_ROOT}"
    if [[ "${DO_PUSH}" == "1" ]]; then
        run_cmd docker push "${ACCOUNT_IMAGE}"
    fi
fi

# ---------- 构建 iam-front ----------
if [[ -z "${ONLY}" || "${ONLY}" == "front" ]]; then
    log_step 4 4 "构建 iam-front 镜像"
    run_cmd docker build \
        -t "${FRONT_IMAGE}" \
        -f "${PROJECT_ROOT}/dockerfile/frontend/iam-front/Dockerfile" \
        "${PROJECT_ROOT}"
    if [[ "${DO_PUSH}" == "1" ]]; then
        run_cmd docker push "${FRONT_IMAGE}"
    fi
fi

# ---------- 记录 tag，供 deploy-* 复用 ----------
save_image_tag "${PROJECT_ROOT}" "${IMAGE_TAG}"

log_section "构建完成"
if [[ -z "${ONLY}" || "${ONLY}" == "account" ]]; then
    log_ok "iam-account: ${ACCOUNT_IMAGE}"
fi
if [[ -z "${ONLY}" || "${ONLY}" == "front" ]]; then
    log_ok "iam-front  : ${FRONT_IMAGE}"
fi
echo ""
log_hint "下一步部署时会自动复用该 tag；或显式覆盖： IMAGE_TAG=${IMAGE_TAG} bash deploy/scripts/deploy-iam.sh"
