#!/usr/bin/env bash
# =============================================================================
# deploy/scripts/preflight.sh
# -----------------------------------------------------------------------------
# 用途：把 how-dev-iam 部署到本机 Colima k3s 集群前的**环境体检**脚本。
#
# 行为约定（与团队部署规范一致）：
#   1) 只检测，不修改环境。所有缺失项打印**可复制的修复命令**，由使用者手动执行。
#   2) 检查完成后打印汇总；存在阻塞项则以退出码 1 结束。
#   3) 幂等，随时可以重复执行。
#
# 使用：
#   bash deploy/scripts/preflight.sh
#   bash deploy/scripts/preflight.sh --kubeconfig ~/.kube/prod   # 指定 kubeconfig
#
# 环境变量（可选）：
#   REGISTRY_HOST     宿主机侧私有 registry 地址，默认 localhost:5000
#   REGISTRY_NODE     k8s 节点侧拉取地址，       默认 host.docker.internal:5000
#   REGISTRY_USER     私有 registry 用户名，        默认 admin
#   REGISTRY_PASSWORD 私有 registry 密码，          默认 admin123
#   COLIMA_PROFILE    Colima profile 名，        默认 k8s
#   IAM_HOST          Ingress host，              默认 iam.local
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
REGISTRY_USER="${REGISTRY_USER:-admin}"
REGISTRY_PASSWORD="${REGISTRY_PASSWORD:-admin123}"
COLIMA_PROFILE="${COLIMA_PROFILE:-k8s}"
IAM_HOST="${IAM_HOST:-iam.local}"

# ---------- 参数解析 ----------
KUBECONFIG_FILE=""
usage() {
    cat <<EOF
用法：$(basename "$0") [选项]

选项：
  --kubeconfig <path>    指定 kubeconfig 文件路径（未指定时使用默认 \$KUBECONFIG 或 ~/.kube/config）
                         支持绝对/相对路径与 ~ 展开
  -h, --help             显示帮助
EOF
}
while [[ $# -gt 0 ]]; do
    case "$1" in
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
KUBECONFIG_SRC="$(kubeconfig_source)"

log_section "how-dev-iam 部署前置检查"
log_info "项目根路径      : ${PROJECT_ROOT}"
log_info "Registry (host) : ${REGISTRY_HOST}"
log_info "Registry (node) : ${REGISTRY_NODE}"
log_info "Registry 用户   : ${REGISTRY_USER}"
log_info "Colima profile  : ${COLIMA_PROFILE}"
log_info "Ingress host    : ${IAM_HOST}"
log_info "Kubeconfig      : ${KUBECONFIG:-~/.kube/config}  (来源：${KUBECONFIG_SRC})"

# =============================================================================
# 1. CLI 工具
# =============================================================================
log_section "1) CLI 工具"

check_cli() {
    local cmd="$1" install_hint="$2"
    if have_cmd "${cmd}"; then
        check_pass "${cmd} 已安装：$(command -v "${cmd}")"
    else
        check_fail "缺少 ${cmd}"
        log_fix "请手动安装 ${cmd}：" "${install_hint}"
    fi
}

check_cli docker  "在 https://www.docker.com/products/docker-desktop 下载并安装 Docker Desktop（或使用 colima 提供的 docker）"
check_cli colima  "brew install colima"
check_cli kubectl "brew install kubectl"
check_cli helm    "brew install helm"
check_cli make    "xcode-select --install    # 若已安装可跳过"

# =============================================================================
# 2. Docker daemon 可用
# =============================================================================
log_section "2) Docker daemon"

if have_cmd docker; then
    if docker info >/dev/null 2>&1; then
        check_pass "docker daemon 可用：$(docker --version)"
    else
        check_fail "docker 命令存在但 daemon 未运行"
        log_fix "启动 Docker Desktop 或 Colima：" \
                "open -a Docker    # Docker Desktop" \
                "colima start      # 或使用 colima"
    fi
else
    log_hint "docker 未安装，跳过 daemon 检查"
fi

# =============================================================================
# 3. Colima k8s profile
# =============================================================================
log_section "3) Colima k8s profile: ${COLIMA_PROFILE}"

if have_cmd colima; then
    # colima list 输出格式：PROFILE STATUS ARCH CPUS MEMORY DISK RUNTIME ADDRESS
    colima_status="$(colima list 2>/dev/null | awk -v p="${COLIMA_PROFILE}" '$1==p {print $2; exit}')"
    if [[ -z "${colima_status}" ]]; then
        check_fail "Colima profile '${COLIMA_PROFILE}' 不存在"
        log_fix "创建并启动 k8s profile（首次执行；参数按需调整）：" \
                "colima start ${COLIMA_PROFILE} --kubernetes --cpu 4 --memory 8 --disk 100"
    elif [[ "${colima_status}" != "Running" ]]; then
        check_fail "Colima profile '${COLIMA_PROFILE}' 状态为 ${colima_status}"
        log_fix "启动 profile：" \
                "colima start ${COLIMA_PROFILE}"
    else
        check_pass "Colima profile '${COLIMA_PROFILE}' 状态：Running"
    fi
else
    log_hint "colima 未安装，跳过 profile 检查"
fi

# =============================================================================
# 4. k8s 集群 & 上下文
# =============================================================================
log_section "4) Kubernetes 集群"

if have_cmd kubectl; then
    if kubectl version --request-timeout=5s >/dev/null 2>&1 && kubectl cluster-info --request-timeout=5s >/dev/null 2>&1; then
        current_ctx="$(kubectl config current-context 2>/dev/null || echo '<none>')"
        check_pass "kubectl 可以访问集群，当前 context = ${current_ctx}"

        # 节点 Ready
        not_ready="$(kubectl get nodes --no-headers 2>/dev/null | awk '$2!="Ready"{print $1}')"
        if [[ -n "${not_ready}" ]]; then
            check_fail "存在非 Ready 状态的节点：${not_ready}"
            log_fix "查看节点状态并等待就绪：" \
                    "kubectl get nodes -o wide" \
                    "kubectl describe node <node-name>"
        else
            node_count="$(kubectl get nodes --no-headers 2>/dev/null | wc -l | tr -d ' ')"
            check_pass "所有 ${node_count} 个节点均为 Ready"
        fi
    else
        check_fail "kubectl 无法访问集群（超时或未配置 kubeconfig）"
        log_fix "确认 Colima k8s profile 正在运行，并且 kubeconfig 指向它：" \
                "colima start ${COLIMA_PROFILE}" \
                "kubectl config get-contexts" \
                "kubectl config use-context colima-${COLIMA_PROFILE}"
    fi
else
    log_hint "kubectl 未安装，跳过集群检查"
fi

# =============================================================================
# 5. 本地私有 Docker Registry（支持 Basic Auth）
# =============================================================================
log_section "5) 本地私有 Docker Registry (${REGISTRY_HOST})"

reg_host="${REGISTRY_HOST%%:*}"
reg_port="${REGISTRY_HOST##*:}"

if port_reachable "${reg_host}" "${reg_port}" 2; then
    if registry_online "${REGISTRY_HOST}"; then
        # 能得到 200 或 401，都说明 registry 在线
        if registry_auth_required "${REGISTRY_HOST}"; then
            check_pass "Registry 在线且开启了 Basic Auth：http://${REGISTRY_HOST}/v2/"
            if registry_credentials_ok "${REGISTRY_HOST}" "${REGISTRY_USER}" "${REGISTRY_PASSWORD}"; then
                check_pass "Registry 凭据可用（user=${REGISTRY_USER}）"
            else
                check_fail "Registry 凭据无效（user=${REGISTRY_USER}）"
                log_fix "确认环境变量 REGISTRY_USER / REGISTRY_PASSWORD 与本地 registry 匹配：" \
                        "curl -sI -u \"${REGISTRY_USER}:******\" http://${REGISTRY_HOST}/v2/" \
                        "# 默认凭据为 admin/admin123；若你修改过 install.sh 里的密码，请相应 export"
            fi
        else
            check_pass "Registry v2 API 可访问且未开认证：http://${REGISTRY_HOST}/v2/"
        fi
    else
        check_warn "端口 ${REGISTRY_HOST} 可达但 /v2/ 未返回 200/401，可能不是 registry:2"
        log_fix "确认本地 registry 正常：" \
                "curl -v http://${REGISTRY_HOST}/v2/" \
                "docker ps --filter name=docker-registry-local"
    fi
else
    check_fail "无法连接本地 registry：${REGISTRY_HOST}"
    log_fix "启动 how-dev-cloud-native 项目提供的本地 registry：" \
            "bash /Users/zefinnhao/SoftwareDevelop/projects/hzf_project/how-dev-cloud-native/infrastructure/Mac_docker_install_dockerhub/install.sh"
fi

# ---------- 宿主机 docker login 状态 ----------
# 只有在 registry 开启认证时才需要 login；本地开发下 build.sh 会自动登录，这里只提醒。
if have_cmd docker && docker info >/dev/null 2>&1 && registry_auth_required "${REGISTRY_HOST}"; then
    docker_cfg="${DOCKER_CONFIG:-${HOME}/.docker}/config.json"
    if [[ -f "${docker_cfg}" ]] && grep -q "\"${REGISTRY_HOST}\"" "${docker_cfg}" 2>/dev/null; then
        check_pass "宿主机 docker 已登录 ${REGISTRY_HOST}（${docker_cfg}）"
    else
        check_warn "宿主机 docker 未登录 ${REGISTRY_HOST}（build.sh 会自动登录，不阻塞部署）"
        log_fix "可预先手动登录（使用 --password-stdin 避免明文）：" \
                "echo '${REGISTRY_PASSWORD}' | docker login ${REGISTRY_HOST} -u ${REGISTRY_USER} --password-stdin"
    fi
fi

# =============================================================================
# 6. Colima VM 的 docker daemon 是否信任 http registry
# =============================================================================
log_section "6) Colima VM insecure-registry 配置"

if have_cmd colima && [[ "${colima_status:-}" == "Running" ]]; then
    # 抓取 VM 内 daemon.json；不存在时视为不合规
    vm_daemon_json="$(colima ssh -p "${COLIMA_PROFILE}" -- cat /etc/docker/daemon.json 2>/dev/null || echo '{}')"
    if echo "${vm_daemon_json}" | grep -q "${REGISTRY_NODE}"; then
        check_pass "VM 内 daemon.json 已包含 insecure-registry: ${REGISTRY_NODE}"
    else
        check_fail "VM 内 daemon.json 未把 ${REGISTRY_NODE} 加入 insecure-registries"
        log_fix "在 Colima VM 中修改 /etc/docker/daemon.json 并重启 docker：" \
                "colima ssh -p ${COLIMA_PROFILE} -- sudo sh -c 'python3 -c \"import json,os; p=\\\"/etc/docker/daemon.json\\\"; d=json.load(open(p)) if os.path.exists(p) else {}; d[\\\"insecure-registries\\\"]=list(set(d.get(\\\"insecure-registries\\\",[])+[\\\"${REGISTRY_NODE}\\\",\\\"${REGISTRY_HOST}\\\"])); open(p,\\\"w\\\").write(json.dumps(d,indent=2))\" && systemctl restart docker'" \
                "colima ssh -p ${COLIMA_PROFILE} -- cat /etc/docker/daemon.json   # 验证"
    fi

    # 顺便测试一下 VM 是否真的能访问 registry（网络层连通性）
    # 注意：curl 通过不代表 docker daemon 也允许 pull，daemon 是否放行仍取决于 insecure-registries 配置。
    if colima ssh -p "${COLIMA_PROFILE}" -- curl -fsS --max-time 3 "http://${REGISTRY_NODE}/v2/" >/dev/null 2>&1; then
        check_pass "Colima VM 网络层可访问 ${REGISTRY_NODE}（curl 探测通过；docker pull 是否允许仍由上一项决定）"
    else
        check_warn "Colima VM 网络层无法访问 http://${REGISTRY_NODE}/v2/（若刚起 registry 可稍后再试）"
        log_fix "手动在 VM 内验证连通性：" \
                "colima ssh -p ${COLIMA_PROFILE} -- curl -v http://${REGISTRY_NODE}/v2/"
    fi
else
    log_hint "Colima VM 未在运行，跳过 daemon.json / VM 连通性检查"
fi

# =============================================================================
# 7. Ingress Controller (ingress-nginx)
# =============================================================================
log_section "7) Ingress Controller"

if have_cmd kubectl && kubectl cluster-info --request-timeout=5s >/dev/null 2>&1; then
    if kubectl get ns ingress-nginx >/dev/null 2>&1; then
        # 按 label 查找 controller Deployment（兼容任意 helm release name）
        deploy_name="$(resolve_ingress_deploy_name)"
        if [[ -z "${deploy_name}" ]]; then
            check_fail "ingress-nginx 命名空间存在，但未找到 controller Deployment（按 label 查找）"
            log_fix "确认基础设施已正确部署：" \
                    "kubectl -n ingress-nginx get deploy -l app.kubernetes.io/name=ingress-nginx" \
                    "bash deploy/scripts/deploy-infra.sh"
        else
            ready="$(kubectl -n ingress-nginx get deploy "${deploy_name}" \
                        -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)"
            desired="$(kubectl -n ingress-nginx get deploy "${deploy_name}" \
                        -o jsonpath='{.spec.replicas}' 2>/dev/null || echo 0)"
            if [[ "${ready:-0}" -gt 0 && "${ready}" == "${desired}" ]]; then
                check_pass "ingress-nginx controller (${deploy_name}) Ready (${ready}/${desired})"
            else
                check_fail "ingress-nginx controller (${deploy_name}) 未就绪 (${ready:-0}/${desired:-0})"
                log_fix "查看/等待就绪：" \
                        "kubectl -n ingress-nginx get pods" \
                        "kubectl -n ingress-nginx wait --for=condition=Available deploy/${deploy_name} --timeout=180s"
            fi
        fi
    else
        check_fail "集群未安装 ingress-nginx"
        log_fix "部署基础公共组件（k8s-infra chart 会自动装 ingress-nginx）：" \
                "bash deploy/scripts/deploy-infra.sh"
    fi

    # ingressclass
    if kubectl get ingressclass nginx >/dev/null 2>&1; then
        check_pass "IngressClass 'nginx' 存在"
    else
        check_warn "未找到 IngressClass 'nginx'（若刚装完可稍等）"
        log_fix "查看当前 IngressClass：" \
                "kubectl get ingressclass"
    fi
else
    log_hint "集群不可用，跳过 ingress 检查"
fi

# =============================================================================
# 8. /etc/hosts 是否有 IAM_HOST 映射
# =============================================================================
log_section "8) /etc/hosts 映射"

if grep -qE "^\s*[^#]*\s${IAM_HOST}(\s|$)" /etc/hosts 2>/dev/null; then
    check_pass "/etc/hosts 已包含 ${IAM_HOST} 映射"
else
    check_fail "/etc/hosts 缺少 ${IAM_HOST} 映射"
    log_fix "追加 host 映射（需要 sudo）：" \
            "echo '127.0.0.1 ${IAM_HOST}' | sudo tee -a /etc/hosts"
fi

# =============================================================================
# 9. 端口占用检查（80/443 供 ingress 使用）
# =============================================================================
log_section "9) 宿主机端口占用"

check_port_free() {
    local port="$1" hint="$2"
    if lsof -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
        occupant="$(lsof -nP -iTCP:"${port}" -sTCP:LISTEN | awk 'NR==2{print $1"(pid "$2")"}')"
        check_warn "宿主机端口 ${port} 已被占用：${occupant}"
        log_fix "${hint}" \
                "lsof -nP -iTCP:${port} -sTCP:LISTEN" \
                "# 或者改用 kubectl port-forward 方案，见 deploy/README.md"
    else
        check_pass "宿主机端口 ${port} 空闲"
    fi
}
check_port_free 80  "释放 80 端口，或后续通过 kubectl port-forward 走高位端口访问 ingress"
check_port_free 443 "释放 443 端口（若不启用 TLS 可忽略）"

# =============================================================================
# 10. 项目 Chart / Dockerfile 完整性
# =============================================================================
log_section "10) 项目产物完整性"

for f in \
    "${PROJECT_ROOT}/deploy/k8s/helm/middleware/Chart.yaml" \
    "${PROJECT_ROOT}/deploy/k8s/helm/init-jobs/Chart.yaml" \
    "${PROJECT_ROOT}/deploy/k8s/helm/k8s-infra/Chart.yaml" \
    "${PROJECT_ROOT}/deploy/k8s/helm/iam/Chart.yaml" \
    "${PROJECT_ROOT}/deploy/k8s/helm/iam/charts/iam-account/Chart.yaml" \
    "${PROJECT_ROOT}/deploy/k8s/helm/iam/charts/iam-front/Chart.yaml" \
    "${PROJECT_ROOT}/dockerfile/backend/iam-account/Dockerfile" \
    "${PROJECT_ROOT}/dockerfile/frontend/iam-front/Dockerfile"
do
    if [[ -f "${f}" ]]; then
        check_pass "$(realpath --relative-to="${PROJECT_ROOT}" "${f}" 2>/dev/null || echo "${f}")"
    else
        check_fail "缺失文件：${f}"
        log_fix "确认已同步最新代码：" \
                "git -C ${PROJECT_ROOT} status"
    fi
done

# helm lint（若 helm 可用）
if have_cmd helm; then
    for chart in middleware init-jobs k8s-infra iam; do
        if helm lint "${PROJECT_ROOT}/deploy/k8s/helm/${chart}" >/dev/null 2>&1; then
            check_pass "helm lint deploy/k8s/helm/${chart} 通过"
        else
            check_fail "helm lint deploy/k8s/helm/${chart} 失败"
            log_fix "查看详细输出：" \
                    "helm lint ${PROJECT_ROOT}/deploy/k8s/helm/${chart}"
        fi
    done
fi

# =============================================================================
# 汇总
# =============================================================================
log_section "汇总"
summary_and_print
