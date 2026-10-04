#!/usr/bin/env bash
# =============================================================================
# deploy/scripts/deploy-headlamp.sh
# -----------------------------------------------------------------------------
# 用途：部署 Headlamp（Kubernetes Web Dashboard）到本地 k3s 集群，
#      并创建一个 cluster-admin ServiceAccount 用于登录。
#
# 与 iam 业务解耦：独立 namespace（默认 headlamp）、独立 helm release，
# 不进入 deploy-all.sh 一键编排流程；卸载不影响 iam。
#
# 内置最小前置检查：
#   - helm / kubectl 存在、集群可访问
#   - 使用 Ingress 时：ingress-nginx-controller 就绪
#
# 使用：
#   bash deploy/scripts/deploy-headlamp.sh                # 默认：port-forward 访问
#   bash deploy/scripts/deploy-headlamp.sh --with-ingress # 走 Ingress（HEADLAMP_HOST）
#   bash deploy/scripts/deploy-headlamp.sh --port-forward # 前台起 port-forward（Ctrl+C 停）
#   bash deploy/scripts/deploy-headlamp.sh --print-token  # 只打印登录 token 后退出
#   bash deploy/scripts/deploy-headlamp.sh --uninstall    # 一键卸载
#   bash deploy/scripts/deploy-headlamp.sh --kubeconfig ~/.kube/prod   # 指定 kubeconfig
#   bash deploy/scripts/deploy-headlamp.sh --dry-run
#
# 环境变量（可选）：
#   HEADLAMP_NS       部署 namespace（默认 headlamp）
#   HEADLAMP_RELEASE  helm release 名（默认 headlamp）
#   HEADLAMP_SA       登录用的 ServiceAccount 名（默认 headlamp-admin）
#   HEADLAMP_ROLE     绑定的 ClusterRole（默认 cluster-admin；生产可换 view）
#   HEADLAMP_HOST     启用 Ingress 时的 host（默认 headlamp.local）
#   HEADLAMP_PORT     port-forward 时宿主机端口（默认 8090）
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

HEADLAMP_NS="${HEADLAMP_NS:-headlamp}"
HEADLAMP_RELEASE="${HEADLAMP_RELEASE:-headlamp}"
HEADLAMP_SA="${HEADLAMP_SA:-headlamp-admin}"
HEADLAMP_ROLE="${HEADLAMP_ROLE:-cluster-admin}"
HEADLAMP_HOST="${HEADLAMP_HOST:-headlamp.local}"
HEADLAMP_PORT="${HEADLAMP_PORT:-8090}"

# ClusterRoleBinding 名字：拼接 SA 名 + role 名，保证多套 SA/角色时不冲突
HEADLAMP_CRB="${HEADLAMP_SA}-${HEADLAMP_ROLE}"

# ---------- 参数 ----------
DRY_RUN=0
WITH_INGRESS=0
DO_PORT_FORWARD=0
PRINT_TOKEN_ONLY=0
UNINSTALL=0
KUBECONFIG_FILE=""

usage() {
    cat <<EOF
用法：$(basename "$0") [选项]

选项：
  --with-ingress         创建 Ingress（host=\${HEADLAMP_HOST}，默认 headlamp.local）
                         需要 /etc/hosts 已配置：127.0.0.1 \${HEADLAMP_HOST}
  --port-forward         部署完成后前台启动 kubectl port-forward（Ctrl+C 停止）
  --print-token          不部署，只重新生成并打印登录 token
  --uninstall            卸载 headlamp（release / SA / ClusterRoleBinding / namespace）
  --dry-run              只打印命令
  --kubeconfig <path>    指定 kubeconfig 文件路径（未指定时使用默认 \$KUBECONFIG 或 ~/.kube/config）
                         支持绝对/相对路径与 ~ 展开
  -h, --help             显示帮助

环境变量：
  HEADLAMP_NS       部署 namespace         (默认 headlamp)
  HEADLAMP_RELEASE  helm release 名         (默认 headlamp)
  HEADLAMP_SA       登录用 SA 名            (默认 headlamp-admin)
  HEADLAMP_ROLE     绑定的 ClusterRole      (默认 cluster-admin)
  HEADLAMP_HOST     Ingress host            (默认 headlamp.local)
  HEADLAMP_PORT     port-forward 宿主机端口 (默认 8090)
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --with-ingress) WITH_INGRESS=1 ;;
        --port-forward) DO_PORT_FORWARD=1 ;;
        --print-token)  PRINT_TOKEN_ONLY=1 ;;
        --uninstall)    UNINSTALL=1 ;;
        --dry-run)      DRY_RUN=1 ;;
        --kubeconfig)
            KUBECONFIG_FILE="$(parse_kubeconfig_arg "${2:-}")"
            export KUBECONFIG="${KUBECONFIG_FILE}"
            shift
            ;;
        -h|--help)      usage; exit 0 ;;
        *) log_error "未知参数：$1"; usage; exit 2 ;;
    esac
    shift
done
export DRY_RUN
KUBECONFIG_SRC="$(kubeconfig_source)"

# =============================================================================
# 辅助函数
# =============================================================================

# 打印登录 token
# TokenRequest API (kubectl create token) 生成的 token 有有效期，
# 默认 1h。这里请求 24h，方便本地开发一整天不用重登。
print_login_token() {
    local ns="$1" sa="$2"
    if ! kubectl -n "${ns}" get sa "${sa}" >/dev/null 2>&1; then
        log_error "ServiceAccount ${ns}/${sa} 不存在，无法生成 token"
        log_fix "先部署 headlamp：" \
                "bash deploy/scripts/deploy-headlamp.sh"
        return 1
    fi
    local token
    token="$(kubectl -n "${ns}" create token "${sa}" --duration=24h 2>/dev/null || true)"
    if [[ -z "${token}" ]]; then
        log_error "kubectl create token 失败"
        return 1
    fi
    echo ""
    echo -e "${_C_BOLD}${_C_GREEN}=== Headlamp 登录 Token（24h 有效） ===${_C_NC}"
    echo "${token}"
    echo ""
    log_hint "复制上面的 token，粘贴到 Headlamp 登录页 → Bearer Token 输入框。"
    log_hint "过期后重新生成： bash deploy/scripts/deploy-headlamp.sh --print-token"
}

# 幂等地创建/更新 SA 与 ClusterRoleBinding
ensure_sa_and_binding() {
    local ns="$1" sa="$2" role="$3" crb="$4"
    echo -e "  ${_C_DIM}\$${_C_NC} kubectl -n ${ns} apply ServiceAccount/${sa} + ClusterRoleBinding/${crb} (role=${role})"
    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        return 0
    fi
    # SA
    kubectl create serviceaccount "${sa}" -n "${ns}" \
        --dry-run=client -o yaml \
      | kubectl apply -f - >/dev/null
    # ClusterRoleBinding
    kubectl create clusterrolebinding "${crb}" \
        --clusterrole="${role}" \
        --serviceaccount="${ns}:${sa}" \
        --dry-run=client -o yaml \
      | kubectl apply -f - >/dev/null
    log_ok "SA ${ns}/${sa} + ClusterRoleBinding ${crb} 已同步（role=${role}）"
}

# 幂等地创建/更新 Ingress
ensure_ingress() {
    local ns="$1" host="$2" release="$3"
    # headlamp chart 生成的 Service 名等于 release name
    local svc="${release}"
    echo -e "  ${_C_DIM}\$${_C_NC} kubectl -n ${ns} apply Ingress/headlamp (host=${host} → svc/${svc}:80)"
    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        return 0
    fi
    cat <<EOF | kubectl apply -f - >/dev/null
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: headlamp
  namespace: ${ns}
  labels:
    app.kubernetes.io/name: headlamp
    app.kubernetes.io/managed-by: deploy-headlamp.sh
spec:
  ingressClassName: nginx
  rules:
    - host: ${host}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: ${svc}
                port:
                  number: 80
EOF
    log_ok "Ingress ${ns}/headlamp 已同步（host=${host}）"
}

# 卸载：清理 release / SA / CRB / Ingress / ns
do_uninstall() {
    print_env_summary "卸载 Headlamp" \
        "Namespace       : ${HEADLAMP_NS}" \
        "Release         : ${HEADLAMP_RELEASE}" \
        "ServiceAccount  : ${HEADLAMP_SA}" \
        "ClusterRoleBind : ${HEADLAMP_CRB}"

    log_step 1 4 "卸载 helm release"
    if helm -n "${HEADLAMP_NS}" status "${HEADLAMP_RELEASE}" >/dev/null 2>&1; then
        run_cmd helm -n "${HEADLAMP_NS}" uninstall "${HEADLAMP_RELEASE}"
    else
        log_info "release ${HEADLAMP_RELEASE} 不存在，跳过"
    fi

    log_step 2 4 "删除 ClusterRoleBinding"
    if kubectl get clusterrolebinding "${HEADLAMP_CRB}" >/dev/null 2>&1; then
        run_cmd kubectl delete clusterrolebinding "${HEADLAMP_CRB}"
    else
        log_info "clusterrolebinding ${HEADLAMP_CRB} 不存在，跳过"
    fi

    log_step 3 4 "删除 Ingress（若存在）"
    if kubectl -n "${HEADLAMP_NS}" get ingress headlamp >/dev/null 2>&1; then
        run_cmd kubectl -n "${HEADLAMP_NS}" delete ingress headlamp
    else
        log_info "ingress headlamp 不存在，跳过"
    fi

    log_step 4 4 "删除 namespace ${HEADLAMP_NS}"
    if kubectl get ns "${HEADLAMP_NS}" >/dev/null 2>&1; then
        run_cmd kubectl delete ns "${HEADLAMP_NS}"
    else
        log_info "namespace ${HEADLAMP_NS} 不存在，跳过"
    fi

    log_section "Headlamp 卸载完成"
}

# =============================================================================
# 主流程分支
# =============================================================================

# ---- 分支：--uninstall ----
if [[ "${UNINSTALL}" == "1" ]]; then
    require_kubectl_cluster
    require_helm
    do_uninstall
    exit 0
fi

# ---- 分支：--print-token（只重新打印 token） ----
if [[ "${PRINT_TOKEN_ONLY}" == "1" ]]; then
    require_kubectl_cluster
    print_login_token "${HEADLAMP_NS}" "${HEADLAMP_SA}"
    exit 0
fi

# =============================================================================
# 主流程：部署
# =============================================================================

print_env_summary "部署 Headlamp (Kubernetes Dashboard)" \
    "Namespace       : ${HEADLAMP_NS}" \
    "Release         : ${HEADLAMP_RELEASE}" \
    "ServiceAccount  : ${HEADLAMP_SA}   (登录用)" \
    "ClusterRole     : ${HEADLAMP_ROLE}" \
    "启用 Ingress    : $([[ ${WITH_INGRESS} == 1 ]] && echo "yes  (host=${HEADLAMP_HOST})" || echo "no   (使用 port-forward)")" \
    "Kubeconfig      : ${KUBECONFIG:-~/.kube/config}  (来源：${KUBECONFIG_SRC})"

# ---------- 前置检查 ----------
log_step 1 5 "前置检查"
require_helm
require_kubectl_cluster
if [[ "${WITH_INGRESS}" == "1" ]]; then
    require_ingress_controller
    log_ok "ingress-nginx-controller 就绪"
fi
log_ok "前置检查通过"

# ---------- helm repo ----------
log_step 2 5 "配置 helm repo"
# helm repo add 是幂等的（重复添加不会报错，force-update 保证 URL 最新）
run_cmd helm repo add headlamp https://kubernetes-sigs.github.io/headlamp/ --force-update
run_cmd helm repo update headlamp

# ---------- 准备 namespace ----------
log_step 3 5 "准备 namespace ${HEADLAMP_NS}"
if ! kubectl get ns "${HEADLAMP_NS}" >/dev/null 2>&1; then
    run_cmd kubectl create ns "${HEADLAMP_NS}"
else
    log_info "namespace ${HEADLAMP_NS} 已存在"
fi

# ---------- helm upgrade -i ----------
log_step 4 5 "helm upgrade -i ${HEADLAMP_RELEASE}"
# 关键 values：
#   - service.type=ClusterIP：本地默认走 port-forward，不占用 LoadBalancer
#   - 副本数、资源用 chart 默认（本地开发够用）
run_cmd helm upgrade -i "${HEADLAMP_RELEASE}" headlamp/headlamp \
    -n "${HEADLAMP_NS}" \
    --set "config.baseURL=" \
    --set "service.type=ClusterIP" \
    --wait --timeout 5m

# ---------- SA + CRB + (可选) Ingress ----------
log_step 5 5 "创建登录 ServiceAccount 与访问入口"
ensure_sa_and_binding "${HEADLAMP_NS}" "${HEADLAMP_SA}" "${HEADLAMP_ROLE}" "${HEADLAMP_CRB}"

if [[ "${WITH_INGRESS}" == "1" ]]; then
    ensure_ingress "${HEADLAMP_NS}" "${HEADLAMP_HOST}" "${HEADLAMP_RELEASE}"
fi

# =============================================================================
# 完成 + 使用提示
# =============================================================================

log_section "Headlamp 部署完成"

if [[ "${DRY_RUN}" != "1" ]]; then
    echo ""
    echo -e "${_C_BOLD}=== ${HEADLAMP_NS} 命名空间 ===${_C_NC}"
    kubectl get all -n "${HEADLAMP_NS}" 2>/dev/null || true
    if [[ "${WITH_INGRESS}" == "1" ]]; then
        echo ""
        echo -e "${_C_BOLD}=== ${HEADLAMP_NS} Ingress ===${_C_NC}"
        kubectl get ingress -n "${HEADLAMP_NS}" 2>/dev/null || true
    fi

    # 打印 token（放在 kubectl get 之后，避免被资源列表淹没）
    print_login_token "${HEADLAMP_NS}" "${HEADLAMP_SA}"
fi

echo ""
log_ok "访问方式："
if [[ "${WITH_INGRESS}" == "1" ]]; then
    log_hint "  浏览器：http://${HEADLAMP_HOST}"
    log_hint "  前提：/etc/hosts 已有：127.0.0.1 ${HEADLAMP_HOST}"
    log_hint "  兜底：kubectl -n ${HEADLAMP_NS} port-forward svc/${HEADLAMP_RELEASE} ${HEADLAMP_PORT}:80"
else
    log_hint "  在另一个终端执行以下命令启动端口转发："
    log_hint "    kubectl -n ${HEADLAMP_NS} port-forward svc/${HEADLAMP_RELEASE} ${HEADLAMP_PORT}:80"
    log_hint "  然后浏览器打开：http://localhost:${HEADLAMP_PORT}"
    log_hint "  （或本次直接前台起转发： bash deploy/scripts/deploy-headlamp.sh --port-forward）"
fi

echo ""
log_ok "常用命令："
log_hint "  重新打印 token： bash deploy/scripts/deploy-headlamp.sh --print-token"
log_hint "  一键卸载：       bash deploy/scripts/deploy-headlamp.sh --uninstall"
log_hint "  查看 pods：      kubectl -n ${HEADLAMP_NS} get pods"
log_hint "  查看日志：       kubectl -n ${HEADLAMP_NS} logs -l app.kubernetes.io/name=headlamp -f"

# ---------- 可选：前台起 port-forward ----------
if [[ "${DO_PORT_FORWARD}" == "1" && "${DRY_RUN}" != "1" ]]; then
    echo ""
    log_section "启动 port-forward（Ctrl+C 停止）"
    log_hint "浏览器打开：http://localhost:${HEADLAMP_PORT}"
    exec kubectl -n "${HEADLAMP_NS}" port-forward "svc/${HEADLAMP_RELEASE}" "${HEADLAMP_PORT}:80"
fi
