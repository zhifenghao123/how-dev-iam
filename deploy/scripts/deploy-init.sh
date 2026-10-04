#!/usr/bin/env bash
# =============================================================================
# deploy/scripts/deploy-init.sh
# -----------------------------------------------------------------------------
# 用途：执行数据库 schema 初始化 Job（init-jobs chart）。
#   - 通过 helm hook 触发 mysql-schema 建表 Job
#   - 目标 mysql 为 middleware-mysql（跨 ns 用 FQDN）
#
# 内置最小前置检查：
#   - helm / kubectl 存在、集群可访问
#   - middleware-mysql Service 已存在且有 endpoint
#
# 使用：
#   bash deploy/scripts/deploy-init.sh
#   bash deploy/scripts/deploy-init.sh --dry-run
#   bash deploy/scripts/deploy-init.sh --kubeconfig ~/.kube/prod   # 指定 kubeconfig
#
# 环境变量：
#   IAM_NS   业务 ns（默认 iam）
#   MW_NS    中间件 ns（默认 iam-middleware）
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
MW_NS="${MW_NS:-iam-middleware}"
CHART_INIT="${PROJECT_ROOT}/deploy/k8s/helm/init-jobs"

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
print_env_summary "初始化数据库 Schema (init-jobs)" \
    "Chart 路径     : ${CHART_INIT}" \
    "运行 ns        : ${IAM_NS}" \
    "目标 mysql svc : middleware-mysql.${MW_NS}.svc.cluster.local" \
    "Kubeconfig     : ${KUBECONFIG:-~/.kube/config}  (来源：${KUBECONFIG_SRC})"

# ---------- 前置检查 ----------
log_step 1 3 "前置检查"
require_helm
require_kubectl_cluster
# 校验上游 middleware-mysql 已就绪，避免 Job 起来就连接超时
require_svc_ready "${MW_NS}" middleware-mysql 60
log_ok "middleware-mysql 就绪"

# ---------- namespace ----------
log_step 2 3 "准备 namespace ${IAM_NS}"
if ! kubectl get ns "${IAM_NS}" >/dev/null 2>&1; then
    run_cmd kubectl create ns "${IAM_NS}"
else
    log_info "namespace ${IAM_NS} 已存在"
fi

# ---------- helm upgrade -i ----------
log_step 3 3 "helm upgrade -i iam-init"
run_cmd helm upgrade -i iam-init "${CHART_INIT}" \
    -n "${IAM_NS}" \
    --set "mysqlSchema.mysql.host=middleware-mysql.${MW_NS}.svc.cluster.local" \
    --wait --timeout 5m

log_section "Schema 初始化完成"

if [[ "${DRY_RUN}" != "1" ]]; then
    kubectl -n "${IAM_NS}" get jobs 2>/dev/null || true
fi

echo ""
log_ok "接下来："
log_hint "  bash deploy/scripts/deploy-iam.sh"
