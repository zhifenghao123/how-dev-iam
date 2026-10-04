#!/usr/bin/env bash
# =============================================================================
# deploy/scripts/deploy-iam.sh
# -----------------------------------------------------------------------------
# 用途：部署 iam 业务（iam-account + iam-front）。
#
# 内置最小前置检查（fail-fast）：
#   - helm / kubectl 存在、集群可访问
#   - 目标镜像已存在（registry v2 manifest；已适配 Basic Auth）
#   - middleware-mysql Service 已就绲
#   - ingress-nginx-controller Ready
#
# 同时会自动在 iam ns 下创建/更新 imagePullSecret（名为 image-credentials），
# helm chart 通过 global.imagePullSecrets 引用。
#
# 使用：
#   bash deploy/scripts/deploy-iam.sh
#   bash deploy/scripts/deploy-iam.sh --dry-run
#   bash deploy/scripts/deploy-iam.sh --skip-image-check   # 跳过镜像存在性校验
#   bash deploy/scripts/deploy-iam.sh --kubeconfig ~/.kube/prod   # 指定 kubeconfig
#
# 环境变量：
#   REGISTRY_HOST     宿主机地址（默认 localhost:5000）
#   REGISTRY_NODE     节点侧地址（默认 host.docker.internal:5000）
#   REGISTRY_ORG      镜像组织路径（默认 how-dev-iam）
#   IMAGE_TAG         镜像 tag。未指定时优先读取 build.sh 上一次写入的
#                     deploy/.deploy/last-image-tag；若该文件不存在则回退到 "local"。
#                     也可手动传入覆盖： IMAGE_TAG=xxx bash deploy-iam.sh
#   REGISTRY_USER     私有 registry 用户名（默认 admin）
#   REGISTRY_PASSWORD 私有 registry 密码（默认 admin123）
#   IMAGE_PULL_SECRET pull secret 名称（默认 image-credentials）
#   IAM_NS / MW_NS    业务 / 中间件 ns
# =============================================================================

# ---------- 强制 bash 执行 ----------
# macOS 下 `sh xxx.sh` 会强制 POSIX 模式，导致 echo -e / [[ ]] / 数组失效。
# 注意：sh 会设置 POSIXLY_CORRECT=y 并传给新进程，必须先 unset 再 re-exec。
if [ -z "${BASH_VERSION:-}" ]; then
    unset POSIXLY_CORRECT
    exec /usr/bin/env bash "$0" "$@"
fi
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
REGISTRY_NODE="${REGISTRY_NODE:-host.docker.internal:5000}"
REGISTRY_ORG="${REGISTRY_ORG:-how-dev-iam}"
REGISTRY_USER="${REGISTRY_USER:-admin}"
REGISTRY_PASSWORD="${REGISTRY_PASSWORD:-admin123}"
IMAGE_PULL_SECRET="${IMAGE_PULL_SECRET:-image-credentials}"
IAM_NS="${IAM_NS:-iam}"
MW_NS="${MW_NS:-iam-middleware}"

# IMAGE_TAG 解析：显式传入优先；否则优先读取 build.sh 写入的 last-image-tag；再回退 local
if [[ -z "${IMAGE_TAG:-}" ]]; then
    if _tag_from_file="$(load_image_tag "${PROJECT_ROOT}")"; then
        IMAGE_TAG="${_tag_from_file}"
        IMAGE_TAG_SRC="deploy/.deploy/last-image-tag"
    else
        IMAGE_TAG="local"
        IMAGE_TAG_SRC="fallback (未找到 last-image-tag)"
    fi
else
    IMAGE_TAG_SRC="env IMAGE_TAG"
fi

CHART_IAM="${PROJECT_ROOT}/deploy/k8s/helm/iam"

# ---------- 参数 ----------
DRY_RUN=0
SKIP_IMAGE_CHECK=0
KUBECONFIG_FILE=""

usage() {
    cat <<EOF
用法：$(basename "$0") [选项]

选项：
  --skip-image-check     跳过 registry 镜像存在性检查
  --dry-run              只打印命令
  --kubeconfig <path>    指定 kubeconfig 文件路径（未指定时使用默认 \$KUBECONFIG 或 ~/.kube/config）
                         支持绝对路径、相对路径与 ~ 展开：
                           --kubeconfig /Users/foo/.kube/prod        # 绝对路径
                           --kubeconfig ~/.kube/prod                 # 家目录（~ 会自动展开）
                           --kubeconfig ./deploy/.kube/dev           # 相对当前工作目录
                           --kubeconfig ../secrets/kubeconfig-uat    # 上级目录
  -h, --help             显示帮助
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --skip-image-check) SKIP_IMAGE_CHECK=1 ;;
        --dry-run)          DRY_RUN=1 ;;
        --kubeconfig)
            KUBECONFIG_FILE="$(parse_kubeconfig_arg "${2:-}")"
            export KUBECONFIG="${KUBECONFIG_FILE}"
            shift
            ;;
        -h|--help)          usage; exit 0 ;;
        *) log_error "未知参数：$1"; usage; exit 2 ;;
    esac
    shift
done
export DRY_RUN

# ---------- kubeconfig 来源标注（parse_kubeconfig_arg 已完成校验/展开/export） ----------
KUBECONFIG_SRC="$(kubeconfig_source)"

# ---------- 检查镜像是否已推送 ----------
# registry v2：HEAD /v2/<name>/manifests/<tag>。
# 关键点：必须**同时**接受以下 4 类 media type，否则 buildkit / docker buildx
# 生成的多平台镜像（arm64 + amd64 → OCI index / docker manifest list）会命中
# registry 的 "OCI index found, but accept header does not support OCI indexes"
# 错误并返回 404（即便 tag 实际存在）。
# 参考：https://github.com/opencontainers/image-spec/blob/main/media-types.md
check_image_pushed() {
    local repo="$1" tag="$2"
    local url="http://${REGISTRY_HOST}/v2/${repo}/manifests/${tag}"
    curl -fsSI -o /dev/null \
        -u "${REGISTRY_USER}:${REGISTRY_PASSWORD}" \
        -H "Accept: application/vnd.docker.distribution.manifest.v2+json" \
        -H "Accept: application/vnd.docker.distribution.manifest.list.v2+json" \
        -H "Accept: application/vnd.oci.image.manifest.v1+json" \
        -H "Accept: application/vnd.oci.image.index.v1+json" \
        --max-time 5 \
        "${url}"
}

# ---------- 打印计划 ----------
print_env_summary "部署 iam 业务 (iam-account + iam-front)" \
    "Chart 路径      : ${CHART_IAM}" \
    "Namespace       : ${IAM_NS}" \
    "MW namespace    : ${MW_NS}" \
    "Registry (pull) : ${REGISTRY_NODE}" \
    "Registry 用户   : ${REGISTRY_USER}" \
    "ImagePullSecret : ${IAM_NS}/${IMAGE_PULL_SECRET}" \
    "Image org / tag : ${REGISTRY_ORG} / ${IMAGE_TAG}  (来源：${IMAGE_TAG_SRC})" \
    "Kubeconfig      : ${KUBECONFIG:-~/.kube/config}  (来源：${KUBECONFIG_SRC})"

# ---------- 前置检查 ----------
log_step 1 4 "前置检查"
require_helm
require_kubectl_cluster
require_svc_ready "${MW_NS}" middleware-mysql 60
log_ok "middleware-mysql 就绲"
require_ingress_controller
log_ok "ingress-nginx-controller 就绲"

if [[ "${SKIP_IMAGE_CHECK}" == "1" ]]; then
    log_warn "已跳过镜像存在性检查（--skip-image-check）"
else
    for repo in "${REGISTRY_ORG}/iam-account" "${REGISTRY_ORG}/iam-front"; do
        if check_image_pushed "${repo}" "${IMAGE_TAG}"; then
            log_ok "镜像已在 registry：${repo}:${IMAGE_TAG}"
        else
            log_error "registry 中找不到镜像（或凭据无效）：${REGISTRY_HOST}/${repo}:${IMAGE_TAG}"
            log_fix "先构建 & 推送镜像，确保凭据正确：" \
                    "bash deploy/scripts/build.sh" \
                    "# 或手动校验（buildkit 多平台镜像必须同时接受 index/list media type）：" \
                    "curl -sI -u \"${REGISTRY_USER}:******\" -H 'Accept: application/vnd.oci.image.index.v1+json' -H 'Accept: application/vnd.docker.distribution.manifest.list.v2+json' http://${REGISTRY_HOST}/v2/${repo}/manifests/${IMAGE_TAG}"
            exit 1
        fi
    done
fi

# ---------- namespace ----------
log_step 2 4 "准备 namespace ${IAM_NS}"
if ! kubectl get ns "${IAM_NS}" >/dev/null 2>&1; then
    run_cmd kubectl create ns "${IAM_NS}"
else
    log_info "namespace ${IAM_NS} 已存在"
fi

# ---------- imagePullSecret ----------
log_step 3 4 "同步 imagePullSecret ${IAM_NS}/${IMAGE_PULL_SECRET}"
ensure_pull_secret \
    "${IAM_NS}" "${IMAGE_PULL_SECRET}" \
    "${REGISTRY_NODE}" "${REGISTRY_USER}" "${REGISTRY_PASSWORD}"

# ---------- helm upgrade -i ----------
log_step 4 4 "helm upgrade -i iam"
run_cmd helm upgrade -i iam "${CHART_IAM}" \
    -n "${IAM_NS}" \
    --set "global.image.registry=${REGISTRY_NODE}" \
    --set "global.image.repository=${REGISTRY_ORG}" \
    --set "global.image.tag=${IMAGE_TAG}" \
    --set "global.imagePullSecrets[0].name=${IMAGE_PULL_SECRET}" \
    --wait --timeout 5m

log_section "iam 业务部署完成"

if [[ "${DRY_RUN}" != "1" ]]; then
    echo ""
    echo -e "${_C_BOLD}=== ${IAM_NS} 命名空间 ===${_C_NC}"
    kubectl get all -n "${IAM_NS}" 2>/dev/null || true
    echo ""
    echo -e "${_C_BOLD}=== ${IAM_NS} 命名空间 Ingress ===${_C_NC}"
    kubectl get ingress -n "${IAM_NS}" 2>/dev/null || true
fi

IAM_HOST_HINT="${IAM_HOST:-iam.local}"
echo ""
log_ok "访问方式："
log_hint "  浏览器：http://${IAM_HOST_HINT}     （需 /etc/hosts 有：127.0.0.1 ${IAM_HOST_HINT}）"
log_hint "  兜底：  kubectl -n ingress-nginx port-forward svc/ingress-nginx-controller 8080:80"
log_hint "         然后访问 http://${IAM_HOST_HINT}:8080"
echo ""
log_ok "常用调试："
log_hint "  kubectl -n ${IAM_NS} logs -l app.kubernetes.io/name=iam-account --tail=200 -f"
log_hint "  kubectl -n ${IAM_NS} logs -l app.kubernetes.io/name=iam-front   --tail=200 -f"
