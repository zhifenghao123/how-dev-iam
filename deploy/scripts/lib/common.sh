#!/usr/bin/env bash
# =============================================================================
# deploy/scripts/lib/common.sh
# -----------------------------------------------------------------------------
# 供 preflight.sh / deploy.sh 复用的通用工具函数：
#   - 彩色日志：log_info / log_ok / log_warn / log_error / log_step
#   - 交互确认：confirm
#   - 命令 / 端点探测：have_cmd / port_reachable / http_reachable
#   - 步骤计数：step_start / step_summary
#
# 使用方式：
#   #!/usr/bin/env bash
#   set -euo pipefail
#   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   # shellcheck disable=SC1091
#   source "${SCRIPT_DIR}/lib/common.sh"
# =============================================================================

# ---------- 颜色 ----------
# ANSI SGR 转义序列格式：\033[<参数>m
#   \033        = ESC 字符（八进制 033 / 十六进制 0x1B），转义序列起始符
#   [ ... m     = SGR (Select Graphic Rendition) 控制指令
#   参数 0      = 重置所有样式
#   参数 1/2    = 加粗 / 暗淡
#   参数 30-37  = 前景色（30黑 31红 32绿 33黄 34蓝 35紫 36青 37白）
# 说明：仅当 stdout 连接到真实终端 (-t 1) 时才启用颜色；否则全部置空，
#       避免颜色转义污染日志文件 / 管道下游 / CI 采集系统。
if [[ -t 1 ]]; then
    _C_RED='\033[0;31m'      # 红色前景  —— [ERR ] 错误
    _C_GREEN='\033[0;32m'    # 绿色前景  —— [ OK ] 成功
    _C_YELLOW='\033[0;33m'   # 黄色前景  —— [WARN] 警告 / 交互提示
    _C_BLUE='\033[0;34m'     # 蓝色前景  —— [INFO] 信息 / 步骤标题
    _C_CYAN='\033[0;36m'     # 青色前景  —— 章节分隔线 / 命令示例高亮
    _C_BOLD='\033[1m'        # 加粗      —— 标题、汇总行强调
    _C_DIM='\033[2m'         # 暗淡      —— 次要提示、命令回显前缀 "$"
    _C_NC='\033[0m'          # No Color  —— 重置样式，回归终端默认
else
    # 非终端（重定向到文件 / 管道 / CI 无 tty）：颜色变量全部置空，
    # 让 log_* 输出纯文本，日志文件里不会出现 ^[[0;31m 之类的乱码。
    _C_RED=''       # 红色前景  —— [ERR ] 错误
    _C_GREEN=''     # 绿色前景  —— [ OK ] 成功
    _C_YELLOW=''    # 黄色前景  —— [WARN] 警告 / 交互提示
    _C_BLUE=''      # 蓝色前景  —— [INFO] 信息 / 步骤标题
    _C_CYAN=''      # 青色前景  —— 章节分隔线 / 命令示例高亮
    _C_BOLD=''      # 加粗      —— 标题、汇总行强调
    _C_DIM=''       # 暗淡      —— 次要提示、命令回显前缀 "$"
    _C_NC=''        # No Color  —— 重置样式，回归终端默认
fi

# ---------- 日志 ----------
# 说明：以下 5 个日志函数统一约定 —— 传入的所有参数会被 $* 拼成一条消息文本。
# 参数：
#   $@ / $*  ：要输出的日志消息（可含空格、变量展开），例如 log_info "开始构建 ${img}"
# 输出流：
#   log_info / log_ok / log_warn / log_hint → stdout (fd=1)
#   log_error                              → stderr (fd=2)，便于与正常输出分流

# log_info <消息...>   蓝色 [INFO]  普通提示信息
log_info()  { echo -e "${_C_BLUE}[INFO]${_C_NC}  $*"; }
# log_ok   <消息...>   绿色 [ OK ]  成功 / 已完成
log_ok()    { echo -e "${_C_GREEN}[ OK ]${_C_NC}  $*"; }
# log_warn <消息...>   黄色 [WARN]  警告（不阻塞流程）
log_warn()  { echo -e "${_C_YELLOW}[WARN]${_C_NC}  $*"; }
# log_error <消息...>  红色 [ERR ]  错误（写入 stderr）
log_error() { echo -e "${_C_RED}[ERR ]${_C_NC}  $*" >&2; }
# log_hint <消息...>   暗淡缩进    上一条日志的补充说明，无级别标签

# 打印一个手动修复命令块，缩进 + 高亮
# 用法：
#   log_fix "在 Colima VM 中放行本地 http registry：" \
#           "colima ssh -p k8s -- sudo vi /etc/docker/daemon.json"
log_fix() {
    local title="$1"; shift
    echo -e "${_C_YELLOW}       ↳ 修复建议：${title}${_C_NC}"
    for cmd in "$@"; do
        echo -e "         ${_C_CYAN}\$ ${cmd}${_C_NC}"
    done
}

# 步骤分隔线
# log_step <n> <total> <title>
# 参数：
#   $1 n     = 当前步骤序号（如 2）
#   $2 total = 总步骤数（如 5）
#   $3 title = 步骤标题
# 用法：log_step 2 5 "构建 iam-server 镜像"  → ==> [2/5] 构建 iam-server 镜像
log_step() {
    local n="$1" total="$2" title="$3"
    echo ""
    echo -e "${_C_BOLD}${_C_BLUE}==> [${n}/${total}] ${title}${_C_NC}"
}

# 打印章节分隔线
# log_section <标题...>
# 参数：
#   $@ / $*  = 章节标题（会被 ====== 包围输出，视觉上分块）
log_section() {
    echo ""
    echo -e "${_C_BOLD}${_C_CYAN}================================================================${_C_NC}"
    echo -e "${_C_BOLD}${_C_CYAN}  $*${_C_NC}"
    echo -e "${_C_BOLD}${_C_CYAN}================================================================${_C_NC}"
}

# ---------- 交互 ----------
# confirm "prompt text" [default_yes|default_no]
# 返回 0 = yes；返回 1 = no
# 若环境变量 ASSUME_YES=1，则始终返回 0
# 若环境变量 NON_INTERACTIVE=1 且未 ASSUME_YES，则按 default 返回
confirm() {
    local prompt="$1"
    local default="${2:-default_no}"
    if [[ "${ASSUME_YES:-0}" == "1" ]]; then
        echo -e "${_C_DIM}(ASSUME_YES=1) → yes${_C_NC}"
        return 0
    fi
    if [[ "${NON_INTERACTIVE:-0}" == "1" ]]; then
        if [[ "${default}" == "default_yes" ]]; then
            echo -e "${_C_DIM}(NON_INTERACTIVE=1) → yes${_C_NC}"; return 0
        else
            echo -e "${_C_DIM}(NON_INTERACTIVE=1) → no${_C_NC}"; return 1
        fi
    fi
    local hint
    [[ "${default}" == "default_yes" ]] && hint="[Y/n]" || hint="[y/N]"
    read -r -p "$(echo -e "${_C_YELLOW}${prompt} ${hint} ${_C_NC}")" ans
    if [[ -z "${ans}" ]]; then
        [[ "${default}" == "default_yes" ]] && return 0 || return 1
    fi
    case "${ans}" in
        y|Y|yes|YES) return 0 ;;
        *) return 1 ;;
    esac
}

# ---------- 探测 ----------
# have_cmd <cmd>
# 参数：
#   $1 cmd = 要检测的命令名（如 docker、kubectl、helm）
# 返回：
#   0 = 存在于 PATH； 非 0 = 不存在
# 用法：if have_cmd docker; then ...; fi
have_cmd() { command -v "$1" >/dev/null 2>&1; }

# port_reachable host port [timeout_sec]
# 使用 bash 内置的 /dev/tcp，配合子 shell 的 read 超时。
# 注意：/dev/tcp 本身不支持 timeout 参数，这里用一个短的 read 触发连接尝试。
port_reachable() {
    local host="$1" port="$2" timeout="${3:-2}"
    # 关闭 job control 输出，避免 kill 时打印 "Killed" 诊断信息
    ( exec 3<>"/dev/tcp/${host}/${port}" ) >/dev/null 2>&1 &
    local pid=$!
    local waited=0
    while kill -0 "${pid}" 2>/dev/null; do
        if [[ ${waited} -ge $((timeout * 10)) ]]; then
            # 用 SIGTERM 而不是 SIGKILL，避免 shell 打印诊断信息
            { kill "${pid}" 2>/dev/null; wait "${pid}" 2>/dev/null; } >/dev/null 2>&1
            return 1
        fi
        sleep 0.1
        waited=$((waited + 1))
    done
    wait "${pid}" 2>/dev/null
    return $?
}

# http_reachable url [timeout_sec] [user:password]
# 第三个参数用于 Basic Auth（如私有 registry）；留空则不带凭据
http_reachable() {
    local url="$1" timeout="${2:-3}" auth="${3:-}"
    if [[ -n "${auth}" ]]; then
        curl -fsS --max-time "${timeout}" -u "${auth}" "${url}" >/dev/null 2>&1
    else
        curl -fsS --max-time "${timeout}" "${url}" >/dev/null 2>&1
    fi
}

# registry_online <host:port> [user] [password]
# 私有 registry v2 在线判定：
#   - 未开认证：GET /v2/ 返回 200
#   - 开了认证：GET /v2/ 会返回 401；带凭据后返回 200
# 只要能得到 200 或 401 就认为 registry 在线；网络不通 / 5xx 视为离线。
registry_online() {
    local reg="$1" user="${2:-}" pass="${3:-}"
    local code
    if [[ -n "${user}" && -n "${pass}" ]]; then
        code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 -u "${user}:${pass}" "http://${reg}/v2/" || echo 000)"
    else
        code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 "http://${reg}/v2/" || echo 000)"
    fi
    case "${code}" in
        200|401) return 0 ;;
        *)       return 1 ;;
    esac
}

# registry_auth_required <host:port>
# 是否需要认证：GET /v2/ 无凭据时返回 401 即视为需要认证
registry_auth_required() {
    local reg="$1"
    local code
    code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 "http://${reg}/v2/" || echo 000)"
    [[ "${code}" == "401" ]]
}

# registry_credentials_ok <host:port> <user> <password>
# 用给定凭据访问 /v2/，200 视为正确
registry_credentials_ok() {
    local reg="$1" user="$2" pass="$3"
    local code
    code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 -u "${user}:${pass}" "http://${reg}/v2/" || echo 000)"
    [[ "${code}" == "200" ]]
}

# ---------- 检查结果计数器 ----------
# 用法：
#   _CHECK_TOTAL=0; _CHECK_FAIL=0; _CHECK_WARN=0
#   check_pass "描述" / check_fail "描述" / check_warn "描述"
#   summary_and_exit
_CHECK_TOTAL=0
_CHECK_FAIL=0
_CHECK_WARN=0

# check_pass <描述...>   计一次通过（total+1），输出绿色 [ OK ]
# check_warn <描述...>   计一次警告（total+1, warn+1），输出黄色 [WARN]
# check_fail <描述...>   计一次失败（total+1, fail+1），输出红色 [ERR ]
# 参数：
#   $@ / $*  = 描述文本，将透传给对应的 log_* 函数
check_pass() {
    _CHECK_TOTAL=$((_CHECK_TOTAL + 1))
    log_ok "$*"
}
check_warn() {
    _CHECK_TOTAL=$((_CHECK_TOTAL + 1))
    _CHECK_WARN=$((_CHECK_WARN + 1))
    log_warn "$*"
}
check_fail() {
    _CHECK_TOTAL=$((_CHECK_TOTAL + 1))
    _CHECK_FAIL=$((_CHECK_FAIL + 1))
    log_error "$*"
}

# 打印汇总；有 fail 时返回 1
# summary_and_print
# 参数：无（读取上面的 _CHECK_TOTAL / _CHECK_FAIL / _CHECK_WARN 全局计数）
# 返回：0 = 无 fail； 1 = 存在 fail
summary_and_print() {
    echo ""
    echo -e "${_C_BOLD}检查汇总：${_C_NC} 总计 ${_CHECK_TOTAL} 项，通过 $((_CHECK_TOTAL - _CHECK_FAIL - _CHECK_WARN)) 项，警告 ${_CHECK_WARN} 项，失败 ${_CHECK_FAIL} 项"
    if [[ ${_CHECK_FAIL} -gt 0 ]]; then
        echo -e "${_C_RED}${_C_BOLD}✘ 存在阻塞性问题，请按上面 [ERR] 后的『修复建议』手动执行后再重试。${_C_NC}"
        return 1
    elif [[ ${_CHECK_WARN} -gt 0 ]]; then
        echo -e "${_C_YELLOW}${_C_BOLD}⚠ 存在警告，不阻塞部署，但建议处理。${_C_NC}"
        return 0
    else
        echo -e "${_C_GREEN}${_C_BOLD}✔ 环境检查全部通过，可以开始部署。${_C_NC}"
        return 0
    fi
}

# ---------- 项目根路径工具 ----------
# resolve_project_root <lib_dir>
# 根据 lib 目录位置反推项目根目录
# 参数：
#   $1 lib_dir = 本 common.sh 所在目录（约定为 <root>/deploy/scripts/lib）
# 返回：
#   stdout 输出项目根目录绝对路径
# 用法：
#   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   PROJECT_ROOT="$(resolve_project_root "${SCRIPT_DIR}/lib")"
resolve_project_root() {
    local lib_dir="$1"
    # lib_dir = <root>/deploy/scripts/lib
    (cd "${lib_dir}/../../.." && pwd)
}

# =============================================================================
# 【最小前置检查】require_* 系列
# -----------------------------------------------------------------------------
# 每个子部署脚本（build/deploy-infra/deploy-middleware/...）都会在开始时
# 调用一组 require_*，实现"每一步都自带最小前置检查"的约束。
# 检查不通过时打印 [ERR] + 修复建议，并 exit 1。
# =============================================================================

# 提示去跑完整体检
# hint_preflight
# 参数：无
hint_preflight() {
    log_hint "如需完整体检，请执行： bash deploy/scripts/preflight.sh"
}

# require_cmd <cmd> [install_hint]
# 参数：
#   $1 cmd          = 必须存在的命令名
#   $2 install_hint = 安装命令提示（可选，用于失败时打印修复建议）
# 失败时：打印 [ERR] + 修复建议后 exit 1
require_cmd() {
    local cmd="$1"
    local hint="${2:-}"
    if ! have_cmd "${cmd}"; then
        log_error "缺少 CLI：${cmd}"
        if [[ -n "${hint}" ]]; then
            log_fix "请手动安装 ${cmd}：" "${hint}"
        fi
        hint_preflight
        exit 1
    fi
}

# require_docker_daemon
# 参数：无
# 作用：确保 docker CLI 存在 且 docker daemon 已就绪
require_docker_daemon() {
    require_cmd docker
    if ! docker info >/dev/null 2>&1; then
        log_error "docker daemon 未就绪"
        log_fix "启动 Docker Desktop 或 Colima：" \
                "open -a Docker" \
                "colima start"
        exit 1
    fi
}

# require_kubectl_cluster
# 参数：无
# 作用：确保 kubectl 存在 且 当前 kubeconfig 可访问集群（5s 超时）
require_kubectl_cluster() {
    require_cmd kubectl "brew install kubectl"
    if ! kubectl cluster-info --request-timeout=5s >/dev/null 2>&1; then
        log_error "kubectl 无法访问集群"
        log_fix "确认集群运行 & kubeconfig 正确：" \
                "colima start k8s" \
                "kubectl config get-contexts" \
                "kubectl config use-context colima-k8s"
        exit 1
    fi
}

# parse_kubeconfig_arg <path>
# 参数：
#   $1 path = 用户通过 --kubeconfig 传入的路径（绝对/相对/含 ~ 均可）
# 作用：
#   1) 展开 ~ / ~user（应对用户加引号导致 shell 未做波浪号展开的情况）
#   2) 校验文件存在，不存在则 exit 2
#   3) 规范化为绝对路径，avoid 子进程 cwd 变化导致相对路径失效
#   4) 把规范化后的绝对路径通过 stdout 返回
# 返回：
#   stdout = 最终使用的 kubeconfig 绝对路径
#   exit 2 = 参数为空或文件不存在或目录无法解析
# 说明：
#   本函数**不做 export**。原因：调用方通常使用 `X=$(parse_kubeconfig_arg ...)`
#   命令替换语法，这会在子 shell 中执行，函数内的 export 无法影响父 shell。
#   所以由调用方在拿到返回值后，在**父 shell**里显式 `export KUBECONFIG=...`。
#
#   典型用法（配合 case 分支）：
#     --kubeconfig)
#         KUBECONFIG_FILE="$(parse_kubeconfig_arg "${2:-}")"
#         export KUBECONFIG="${KUBECONFIG_FILE}"
#         shift
#         ;;
parse_kubeconfig_arg() {
    local path="${1:-}"
    if [[ -z "${path}" ]]; then
        log_error "--kubeconfig 需要一个文件路径参数"
        exit 2
    fi

    # 1) 手动展开 ~ / ~user
    case "${path}" in
        "~")      path="${HOME}" ;;
        "~/"*)    path="${HOME}/${path#\~/}" ;;
        "~"*)     path="$(eval echo "${path}")" ;;  # ~user 形式
    esac

    # 2) 存在性校验（展开后）
    if [[ ! -f "${path}" ]]; then
        log_error "指定的 kubeconfig 文件不存在：${path}"
        exit 2
    fi

    # 3) 规范为绝对路径（cd/pwd 对已绝对路径也是幂等的）
    local dir base
    dir="$(cd "$(dirname "${path}")" 2>/dev/null && pwd)" || {
        log_error "无法解析 kubeconfig 目录：$(dirname "${path}")"
        exit 2
    }
    base="$(basename "${path}")"
    path="${dir}/${base}"

    # 4) stdout 返回绝对路径（由调用方在父 shell 中 export）
    echo "${path}"
}

# kubeconfig_source
# 参数：无（读取全局 KUBECONFIG_FILE / KUBECONFIG）
# 作用：返回当前 kubeconfig 的来源描述，用于 env summary 展示
# 返回：
#   stdout = 中文来源字符串
kubeconfig_source() {
    if [[ -n "${KUBECONFIG_FILE:-}" ]]; then
        echo "--kubeconfig 参数"
    elif [[ -n "${KUBECONFIG:-}" ]]; then
        echo "env KUBECONFIG"
    else
        echo "默认 ~/.kube/config"
    fi
}

# require_helm
# 参数：无
# 作用：确保 helm CLI 存在
require_helm() {
    require_cmd helm "brew install helm"
}

# require_registry <host:port> [user] [password]
# 判定私有 registry v2 是否在线；若指定了凭据，会校验凭据是否可用。
# - 未开认证的 registry：GET /v2/ = 200，视为在线
# - 开启认证的 registry：GET /v2/ = 401（无凭据）或 200（带凭据）
require_registry() {
    local reg="$1" user="${2:-}" pass="${3:-}"
    if ! registry_online "${reg}" "${user}" "${pass}"; then
        log_error "本地 registry ${reg} 不可达（网络不通 / 5xx）"
        log_fix "启动本地 registry：" \
                "bash /Users/zefinnhao/SoftwareDevelop/projects/hzf_project/how-dev-cloud-native/infrastructure/Mac_docker_install_dockerhub/install.sh"
        exit 1
    fi
    # 若 registry 开了认证，但没给凭据 / 凭据无效，直接 fail-fast
    if registry_auth_required "${reg}"; then
        if [[ -z "${user}" || -z "${pass}" ]]; then
            log_error "registry ${reg} 开启了 Basic Auth，但未提供 REGISTRY_USER / REGISTRY_PASSWORD"
            log_fix "请设置凭据环境变量后重试（默认值为 admin/admin123）：" \
                    "export REGISTRY_USER=admin  REGISTRY_PASSWORD=admin123"
            exit 1
        fi
        if ! registry_credentials_ok "${reg}" "${user}" "${pass}"; then
            log_error "registry ${reg} 凭据校验失败（user=${user}）"
            log_fix "确认 REGISTRY_USER / REGISTRY_PASSWORD 与本地 registry 匹配：" \
                    "curl -sI -u \"${user}:******\" http://${reg}/v2/"
            exit 1
        fi
    fi
}

# docker_login_registry <host:port> <user> <password>
# 参数：
#   $1 reg  = 私有 registry 地址（host:port，如 127.0.0.1:5000）
#   $2 user = registry 用户名（为空则跳过并 log_warn，不视为错误）
#   $3 pass = registry 密码（为空则跳过）
# 说明：
#   - 用 --password-stdin 避免明文密码出现在命令行 / ps 输出
#   - 已登录时再次调用无副作用（幂等）
#   - DRY_RUN=1 时只打印命令、不实际执行
docker_login_registry() {
    local reg="$1" user="$2" pass="$3"
    if [[ -z "${user}" || -z "${pass}" ]]; then
        log_warn "跳过 docker login：REGISTRY_USER / REGISTRY_PASSWORD 为空"
        return 0
    fi
    echo -e "  ${_C_DIM}\$${_C_NC} echo *** | docker login ${reg} -u ${user} --password-stdin"
    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        return 0
    fi
    # 关掉 docker 的 login 输出（防止 "WARNING! Your password will be stored..." 干扰）
    if ! echo "${pass}" | docker login "${reg}" -u "${user}" --password-stdin >/dev/null 2>&1; then
        log_error "docker login ${reg} 失败"
        log_fix "手动验证凭据：" \
                "echo '${pass}' | docker login ${reg} -u ${user} --password-stdin"
        exit 1
    fi
    log_ok "docker login ${reg} 成功（user=${user}）"
}

# ensure_pull_secret <namespace> <secret-name> <registry-host:port> <user> <password>
# 幂等创建/更新 kubernetes.io/dockerconfigjson 类型的 Secret。
# 用于 pod 拉私有 registry 镜像时的 imagePullSecrets 引用。
#
# 说明：
#   - k3s (cri-dockerd via --docker) 拉镜像时使用 VM 内 docker daemon，
#     若 daemon 侧已 docker login 也能成功；但 Secret 是集群持久化对象，
#     更符合 k8s 语义、跨 CRI 迁移零成本。
#   - 用 `apply -f -` 保证幂等（先 create 冲突时 replace 会丢 label 等元数据）。
ensure_pull_secret() {
    local ns="$1" name="$2" reg="$3" user="$4" pass="$5"
    # 保证 ns 存在（调用方通常已经确保）
    if ! kubectl get ns "${ns}" >/dev/null 2>&1; then
        run_cmd kubectl create ns "${ns}"
    fi
    # 用 create --dry-run=client -o yaml | apply -f - 保证幂等
    echo -e "  ${_C_DIM}\$${_C_NC} kubectl -n ${ns} apply Secret/${name} (docker-registry, server=${reg}, user=${user})"
    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        return 0
    fi
    kubectl create secret docker-registry "${name}" \
        --docker-server="${reg}" \
        --docker-username="${user}" \
        --docker-password="${pass}" \
        --namespace="${ns}" \
        --dry-run=client -o yaml \
      | kubectl apply -f - >/dev/null
    log_ok "imagePullSecret ${ns}/${name} 已同步"
}

# require_ns_ready <namespace>  Namespace 存在
# 参数：
#   $1 namespace = 要求存在的 k8s 命名空间
# 失败时：exit 1（并提示如何创建 / 部署上游依赖）
require_ns_ready() {
    local ns="$1"
    if ! kubectl get ns "${ns}" >/dev/null 2>&1; then
        log_error "namespace ${ns} 不存在"
        log_fix "创建 namespace 或先部署上游依赖：" \
                "kubectl create ns ${ns}"
        exit 1
    fi
}

# require_svc_ready <namespace> <svc-name> [timeout_sec]  Service 存在且 endpoint 非空
# 参数：
#   $1 ns          = 命名空间
#   $2 svc         = Service 名
#   $3 timeout_sec = 等待超时秒数，默认 60
# 作用：轮询直到 Service 存在且 endpoints 非空；超时则 exit 1
require_svc_ready() {
    local ns="$1" svc="$2" timeout="${3:-60}"
    require_ns_ready "${ns}"
    local waited=0
    while [[ ${waited} -lt ${timeout} ]]; do
        if kubectl -n "${ns}" get svc "${svc}" >/dev/null 2>&1; then
            local ep
            ep="$(kubectl -n "${ns}" get endpoints "${svc}" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null || true)"
            if [[ -n "${ep}" ]]; then
                return 0
            fi
        fi
        sleep 2
        waited=$((waited + 2))
    done
    log_error "等待 Service ${ns}/${svc} 就绪超时（${timeout}s）"
    log_fix "检查上游 chart 是否已成功部署：" \
            "kubectl -n ${ns} get svc,pods"
    exit 1
}

# resolve_ingress_deploy_name  按 label 查找 ingress-nginx controller 的实际 Deployment 名
# 参数：无
# 返回：
#   stdout = ingress-nginx controller Deployment 名（找到时）
#   非 0   = 未找到
# 官方 ingress-nginx chart 的 controller 资源都带以下 label（与 release name 无关）：
#   app.kubernetes.io/name=ingress-nginx
#   app.kubernetes.io/component=controller
# 通过 label selector 定位，避免硬编码 `<release>-ingress-nginx-controller` 命名。
resolve_ingress_deploy_name() {
    kubectl -n ingress-nginx get deploy \
        -l app.kubernetes.io/name=ingress-nginx,app.kubernetes.io/component=controller \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null
}

# require_ingress_controller  ingress-nginx controller Deployment Ready
# 参数：无
# 作用：校验 ingress-nginx 已安装且 controller Deployment 的 readyReplicas == replicas
# 失败时：exit 1，并提示重新部署基础设施
# 说明：不硬编码 Deployment 名，通过 label 定位。helm 官方 chart 会把
#       release name 作为前缀（如 `k8s-infra-ingress-nginx-controller`），
#       用 label 查找可以避免和 release 名耦合。
require_ingress_controller() {
    if ! kubectl get ns ingress-nginx >/dev/null 2>&1; then
        log_error "ingress-nginx 未安装"
        log_fix "请先部署基础设施：" \
                "bash deploy/scripts/deploy-infra.sh"
        exit 1
    fi
    local deploy_name
    deploy_name="$(resolve_ingress_deploy_name)"
    if [[ -z "${deploy_name}" ]]; then
        log_error "ingress-nginx 命名空间下未找到 controller Deployment（按 label 查找）"
        log_fix "确认基础设施已正确部署：" \
                "kubectl -n ingress-nginx get deploy -l app.kubernetes.io/name=ingress-nginx" \
                "bash deploy/scripts/deploy-infra.sh"
        exit 1
    fi
    local ready desired
    ready="$(kubectl -n ingress-nginx get deploy "${deploy_name}" \
                -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)"
    desired="$(kubectl -n ingress-nginx get deploy "${deploy_name}" \
                -o jsonpath='{.spec.replicas}' 2>/dev/null || echo 0)"
    if [[ "${ready:-0}" -le 0 || "${ready}" != "${desired}" ]]; then
        log_error "ingress-nginx controller (${deploy_name}) 未就绪 (${ready:-0}/${desired:-0})"
        log_fix "等待就绪或重新部署基础设施：" \
                "kubectl -n ingress-nginx wait --for=condition=Available deploy/${deploy_name} --timeout=180s" \
                "bash deploy/scripts/deploy-infra.sh"
        exit 1
    fi
}

# =============================================================================
# 命令执行辅助
# =============================================================================

# run_cmd <cmd> [args...]
# 参数：
#   $@ = 要执行的命令 + 参数
# 行为：
#   - 始终以 `$ <cmd>` 的形式回显命令
#   - 若环境变量 DRY_RUN=1，则只打印不执行（返回 0）
#   - 否则实际执行传入的命令
run_cmd() {
    echo -e "  ${_C_DIM}\$${_C_NC} $*"
    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        return 0
    fi
    "$@"
}

# print_env_summary <section_title> [line1] [line2] ...
# 参数：
#   $1        = 章节标题（透传给 log_section）
#   $2..$N    = 逐条打印的入参说明，每条一行（透传给 log_info）
# 说明：DRY_RUN=1 时额外追加一条 WARN 提示；始终 return 0，避免
#       与 `set -e` 联动导致调用方脚本无 [ERR] 静默退出。
print_env_summary() {
    log_section "$1"
    shift
    for line in "$@"; do
        log_info "${line}"
    done
    # 注意：这里不能用 `[[ ]] && log_warn ...` 的短路写法，否则当
    # DRY_RUN != 1 时整个表达式返回 1，作为函数最后一条语句会让
    # print_env_summary 的返回值变成 1，配合 `set -e` 会导致
    # 调用方（如 build.sh）在没有任何 [ERR] 输出的情况下静默退出。
    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        log_warn "DRY-RUN 模式：只打印命令，不执行"
    fi
    return 0
}

# =============================================================================
# 【镜像 tag 解析与持久化】
# -----------------------------------------------------------------------------
# 生成规则（优先级从高到低）：
#   1) 显式传入 IMAGE_TAG=xxx              → 直接使用
#   2) 显式传入 TAG=xxx（作为 IMAGE_TAG 别名） → 直接使用
#   3) 显式传入 VERSION=vX.Y.Z              → 派生为 "${VERSION}-<时间戳>"
#   4) 都没传                                → 派生为 "v0.0.1-<时间戳>"
#
# 时间戳格式：yyyymmdd-HHMMSS（用 `date "+%Y%m%d-%H%M%S"` 生成）。
# 例：v0.0.1-20260817-113009
#
# build.sh 生成 tag 后会通过 save_image_tag 落盘到 <project>/deploy/.deploy/last-image-tag，
# 供后续 deploy-iam.sh 通过 load_image_tag 复用，保证 build/deploy 两个进程之间 tag 一致。
# =============================================================================

# 默认基线版本号（未设 VERSION 时使用）
_DEFAULT_VERSION="v0.0.1"

# resolve_image_tag [project_root]
# 参数：
#   $1 project_root = 项目根目录（可选，仅在 deploy 侧作为 fallback 使用）
# 环境变量（优先级从高到低）：
#   IMAGE_TAG  → 直接采用
#   TAG        → 作为 IMAGE_TAG 的别名，直接采用
#   VERSION    → 派生为 "${VERSION}-<时间戳>"
#   （都未设）  → 派生为 "${_DEFAULT_VERSION}-<时间戳>"
# 返回：
#   stdout = 最终 tag 字符串（调用方用 $(resolve_image_tag) 接收）
resolve_image_tag() {
    local project_root="${1:-}"
    # 1) IMAGE_TAG 显式指定
    if [[ -n "${IMAGE_TAG:-}" ]]; then
        echo "${IMAGE_TAG}"
        return 0
    fi
    # 2) TAG 作为 IMAGE_TAG 别名
    if [[ -n "${TAG:-}" ]]; then
        echo "${TAG}"
        return 0
    fi
    # 3/4) 派生 <version>-<时间戳>
    local ver="${VERSION:-${_DEFAULT_VERSION}}"
    local ts
    ts="$(date "+%Y%m%d-%H%M%S")"
    echo "${ver}-${ts}"
}

# _tag_state_file <project_root>
# 参数：
#   $1 project_root = 项目根目录绝对路径
# 返回：
#   stdout = tag 状态文件绝对路径（<root>/deploy/.deploy/last-image-tag）
_tag_state_file() {
    local project_root="$1"
    echo "${project_root}/deploy/.deploy/last-image-tag"
}

# save_image_tag <project_root> <tag>
# 参数：
#   $1 project_root = 项目根目录绝对路径
#   $2 tag          = 要落盘的镜像 tag
# 作用：把本次 build 使用的 tag 写入 <root>/deploy/.deploy/last-image-tag，
#       供后续 deploy-* 脚本通过 load_image_tag 读取，保证 build/deploy tag 一致。
#       DRY_RUN=1 时只回显命令，不实际写文件。
save_image_tag() {
    local project_root="$1" tag="$2"
    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        echo -e "  ${_C_DIM}\$${_C_NC} echo ${tag} > $(_tag_state_file "${project_root}")"
        return 0
    fi
    local file
    file="$(_tag_state_file "${project_root}")"
    mkdir -p "$(dirname "${file}")"
    echo "${tag}" > "${file}"
    log_info "已记录本次镜像 tag 到：${file}"
}

# load_image_tag <project_root>
# 参数：
#   $1 project_root = 项目根目录绝对路径
# 返回：
#   stdout = 上一次 build 落盘的 tag（成功时）
#   非 0   = 状态文件不存在或内容为空（调用方需自行 fallback）
load_image_tag() {
    local project_root="$1"
    local file
    file="$(_tag_state_file "${project_root}")"
    if [[ ! -f "${file}" ]]; then
        return 1
    fi
    local tag
    tag="$(head -n1 "${file}" | tr -d '[:space:]')"
    if [[ -z "${tag}" ]]; then
        return 1
    fi
    echo "${tag}"
}
