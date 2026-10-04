#!/usr/bin/env bash
# =============================================================================
# deploy/scripts/deploy-infra.sh
# -----------------------------------------------------------------------------
# 用途：部署 k8s 基础公共组件（k8s-infra chart）：
#   - ingress-nginx controller（通过 helm dependency 引官方 chart）
#   - iam 平台对外 Ingress（host=IAM_HOST，指向 iam ns 下的 iam-front）
#
# 内置最小前置检查：
#   - helm / kubectl 存在
#   - kubectl 能访问集群
#
# 使用：
#   bash deploy/scripts/deploy-infra.sh
#   bash deploy/scripts/deploy-infra.sh --dry-run
#   bash deploy/scripts/deploy-infra.sh --skip-dep-update   # 跳过 helm dep update
#   bash deploy/scripts/deploy-infra.sh --kubeconfig ~/.kube/prod   # 指定 kubeconfig
#
# 环境变量：
#   IAM_NS         业务 namespace，用于放 iam Ingress（默认 iam）
#   IAM_HOST       Ingress host（默认 iam.local）
#   INFRA_NS       k8s-infra release 所在 ns（默认 ingress-nginx）
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

IAM_NS="${IAM_NS:-iam}"
IAM_HOST="${IAM_HOST:-iam.local}"
INFRA_NS="${INFRA_NS:-ingress-nginx}"

CHART_INFRA="${PROJECT_ROOT}/deploy/k8s/helm/k8s-infra"

# ---------- 参数 ----------
DRY_RUN=0
SKIP_DEP_UPDATE=0
KUBECONFIG_FILE=""

usage() {
    cat <<EOF
用法：$(basename "$0") [选项]

选项：
  --skip-dep-update      跳过 helm dependency update（本地已下载过 ingress-nginx 时可用）
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
        --skip-dep-update) SKIP_DEP_UPDATE=1 ;;
        --dry-run)         DRY_RUN=1 ;;
        --kubeconfig)
            KUBECONFIG_FILE="$(parse_kubeconfig_arg "${2:-}")"
            export KUBECONFIG="${KUBECONFIG_FILE}"
            shift
            ;;
        -h|--help)         usage; exit 0 ;;
        *) log_error "未知参数：$1"; usage; exit 2 ;;
    esac
    shift
done
export DRY_RUN
KUBECONFIG_SRC="$(kubeconfig_source)"

# ---------- 打印计划 ----------
print_env_summary "部署 k8s 基础公共组件 (k8s-infra)" \
    "Chart 路径     : ${CHART_INFRA}" \
    "Infra ns       : ${INFRA_NS}   (ingress-nginx controller 装在这里)" \
    "IAM ns         : ${IAM_NS}     (iam Ingress 资源装在这里)" \
    "Ingress host   : ${IAM_HOST}" \
    "Kubeconfig     : ${KUBECONFIG:-~/.kube/config}  (来源：${KUBECONFIG_SRC})"

# ---------- 前置检查 ----------
log_step 1 4 "前置检查"
require_helm
require_kubectl_cluster
log_ok "前置检查通过"

# ---------- 拉取 helm 依赖 ----------
log_step 2 4 "拉取 helm 依赖（ingress-nginx）"
if [[ "${SKIP_DEP_UPDATE}" == "1" ]]; then
    log_info "已跳过（--skip-dep-update）"
else
    run_cmd helm dependency update "${CHART_INFRA}"
fi

# ---------- 确保 namespace 存在 ----------
log_step 3 4 "准备 namespaces"
# iam ns：Ingress 资源需要放到与 iam-front Service 相同的 namespace
if ! kubectl get ns "${IAM_NS}" >/dev/null 2>&1; then
    run_cmd kubectl create ns "${IAM_NS}"
else
    log_info "namespace ${IAM_NS} 已存在"
fi

# ---------- helm upgrade -i ----------
log_step 4 4 "helm upgrade -i k8s-infra"
run_cmd helm upgrade -i k8s-infra "${CHART_INFRA}" \
    -n "${INFRA_NS}" --create-namespace \
    --set "iamIngress.namespace=${IAM_NS}" \
    --set "iamIngress.host=${IAM_HOST}" \
    --wait --timeout 5m

log_section "基础公共组件部署完成"

if [[ "${DRY_RUN}" != "1" ]]; then
    echo ""
    echo -e "${_C_BOLD}=== ${INFRA_NS} 命名空间 ===${_C_NC}"
    kubectl get all -n "${INFRA_NS}" 2>/dev/null || true
    echo ""
    echo -e "${_C_BOLD}=== ${IAM_NS} 命名空间 Ingress ===${_C_NC}"
    kubectl get ingress -n "${IAM_NS}" 2>/dev/null || true
fi

echo ""
log_ok "接下来："
log_hint "  - 若尚未推镜像：bash deploy/scripts/build.sh"
log_hint "  - 部署中间件： bash deploy/scripts/deploy-middleware.sh"
log_hint "  - 部署业务：   bash deploy/scripts/deploy-iam.sh"
log_hint "  - 或一键：     bash deploy/scripts/deploy-all.sh"
