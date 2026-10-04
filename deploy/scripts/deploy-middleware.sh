#!/usr/bin/env bash
# =============================================================================
# deploy/scripts/deploy-middleware.sh
# -----------------------------------------------------------------------------
# 用途：部署 iam 平台依赖的中间件（mysql / redis）。
#
# 内置最小前置检查：
#   - helm / kubectl 存在、集群可访问
#
# 使用：
#   bash deploy/scripts/deploy-middleware.sh
#   bash deploy/scripts/deploy-middleware.sh --dry-run
#   bash deploy/scripts/deploy-middleware.sh --kubeconfig ~/.kube/prod   # 指定 kubeconfig
#
# 环境变量：
#   MW_NS      中间件 namespace（默认 iam-middleware）
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

MW_NS="${MW_NS:-iam-middleware}"
CHART_MIDDLEWARE="${PROJECT_ROOT}/deploy/k8s/helm/middleware"

# ---------- 参数 ----------
DRY_RUN=0
KUBECONFIG_FILE=""

usage() {
    cat <<EOF
用法：$(basename "$0") [选项]

选项：
  --dry-run              只打印命令
  --kubeconfig <path>    指定 kubeconfig 文件路径（未指定时使用默认 \$KUBECONFIG 或 ~/.kube/config）
                         支持绝对/相对路径与 ~ 展开
  -h, --help             显示帮助
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run) DRY_RUN=1 ;;
        --kubeconfig)
            KUBECONFIG_FILE="$(parse_kubeconfig_arg "${2:-}")"
            export KUBECONFIG="${KUBECONFIG_FILE}"
            shift
            ;;
        -h|--help) usage; exit 0 ;;
        *) log_error "未知参数：$1"; usage; exit 2 ;;
    esac
    shift
done
export DRY_RUN
KUBECONFIG_SRC="$(kubeconfig_source)"

# ---------- 打印计划 ----------
print_env_summary "部署中间件 (mysql / redis)" \
    "Chart 路径 : ${CHART_MIDDLEWARE}" \
    "Namespace  : ${MW_NS}" \
    "Kubeconfig : ${KUBECONFIG:-~/.kube/config}  (来源：${KUBECONFIG_SRC})"

# ---------- 前置检查 ----------
log_step 1 3 "前置检查"
require_helm
require_kubectl_cluster
log_ok "前置检查通过"

# ---------- namespace ----------
log_step 2 3 "准备 namespace ${MW_NS}"
if ! kubectl get ns "${MW_NS}" >/dev/null 2>&1; then
    run_cmd kubectl create ns "${MW_NS}"
else
    log_info "namespace ${MW_NS} 已存在"
fi

# ---------- helm upgrade -i ----------
log_step 3 3 "helm upgrade -i middleware"
run_cmd helm upgrade -i middleware "${CHART_MIDDLEWARE}" \
    -n "${MW_NS}" \
    --wait --timeout 5m

log_section "中间件部署完成"

if [[ "${DRY_RUN}" != "1" ]]; then
    kubectl get all -n "${MW_NS}" 2>/dev/null || true
fi

echo ""
log_ok "接下来："
log_hint "  bash deploy/scripts/deploy-init.sh   # 建库 & 建表"
