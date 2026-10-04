#!/usr/bin/env bash
# =============================================================================
# deploy/scripts/deploy-all.sh
# -----------------------------------------------------------------------------
# 用途：一键顺序执行 deploy-infra → deploy-middleware → deploy-init → deploy-iam。
#
# 该脚本本身**不做**具体动作，只是**按序调用**其他 4 个部署脚本；每个子脚本
# 自带最小前置检查，所以本脚本不重复检查。
#
# 注意：本脚本只做**部署编排**，不负责镜像构建。如需构建 / 推送镜像，
#      请单独执行： bash deploy/scripts/build.sh
#
# 使用：
#   bash deploy/scripts/deploy-all.sh                     # 完整部署流程
#   bash deploy/scripts/deploy-all.sh --from middleware   # 从 middleware 开始
#   bash deploy/scripts/deploy-all.sh --skip init         # 跳过某一步（可多次）
#   bash deploy/scripts/deploy-all.sh --kubeconfig ~/.kube/prod   # 指定 kubeconfig（并透传子脚本）
#   bash deploy/scripts/deploy-all.sh --dry-run           # 打印计划
#
# 步骤名：infra | middleware | init | iam
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

STEPS=(infra middleware init iam)

# macOS 自带 bash 3.2 不支持 declare -A（关联数组），用函数模拟
step_script() {
    case "$1" in
        infra)      echo "deploy-infra.sh" ;;
        middleware) echo "deploy-middleware.sh" ;;
        init)       echo "deploy-init.sh" ;;
        iam)        echo "deploy-iam.sh" ;;
    esac
}
step_title() {
    case "$1" in
        infra)      echo "部署基础公共组件" ;;
        middleware) echo "部署中间件" ;;
        init)       echo "初始化数据库 Schema" ;;
        iam)        echo "部署 iam 业务" ;;
    esac
}

FROM=""
SKIP_LIST=()
DRY_RUN=0
KUBECONFIG_FILE=""

# bash 3.2 下访问空数组会因 set -u 报 unbound variable，加个哨兵
_arr_join() {
    # $1=sep, 剩余为数组元素；数组为空时不报错也不输出
    local sep="$1"; shift
    local out=""
    local i
    for i in "$@"; do
        [[ -z "${out}" ]] && out="${i}" || out="${out}${sep}${i}"
    done
    printf '%s' "${out}"
}

usage() {
    cat <<EOF
用法：$(basename "$0") [选项]

本脚本仅负责部署编排（infra / middleware / init / iam），不涉及镜像构建。
镜像构建请单独执行： bash deploy/scripts/build.sh

选项：
  --from <step>          从指定步骤开始执行（step: infra|middleware|init|iam）
  --skip <step>          跳过指定步骤（可多次指定）
  --dry-run              只打印将要执行的子脚本命令
  --kubeconfig <path>    指定 kubeconfig 文件路径，并透传给每个子脚本
                         支持绝对/相对路径与 ~ 展开，例：
                           --kubeconfig ~/.kube/prod
                           --kubeconfig ./deploy/.kube/dev
  -h, --help             显示帮助

示例：
  bash $(basename "$0")                              # 完整部署
  bash $(basename "$0") --from middleware            # 从 middleware 开始
  bash $(basename "$0") --skip init                  # 跳过 init（DDL 已建）
  bash $(basename "$0") --skip middleware --skip init # 只跑基础设施 + 业务
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --from)    FROM="${2:-}"; shift ;;
        --skip)    SKIP_LIST+=("${2:-}"); shift ;;
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
#  DRY_RUN 被 export，让子脚本 / common.sh 里的 run_cmd 能读到
export DRY_RUN
# KUBECONFIG 已在 parse_kubeconfig_arg 中 export，子脚本会自动继承环境变量；
# 为了使用体验与子脚本一致（env summary 中的“来源”展示），
# 若本层显式传了 --kubeconfig，后面仍然会把它作为参数透传给子脚本。
KUBECONFIG_SRC="$(kubeconfig_source)"

# ---------- 校验参数 ----------
step_valid() {
    local s="$1"
    for x in "${STEPS[@]}"; do
        [[ "${x}" == "${s}" ]] && return 0
    done
    return 1
}

if [[ -n "${FROM}" ]] && ! step_valid "${FROM}"; then
    log_error "--from 不识别的步骤：${FROM}"; exit 2
fi
for s in "${SKIP_LIST[@]:-}"; do
    [[ -z "${s}" ]] && continue
    if ! step_valid "${s}"; then log_error "--skip 不识别的步骤：${s}"; exit 2; fi
done

is_skipped() {
    local s="$1"
    for x in "${SKIP_LIST[@]:-}"; do
        [[ -z "${x}" ]] && continue
        [[ "${x}" == "${s}" ]] && return 0
    done
    return 1
}

# ---------- 计算最终计划 ----------
PLAN=()
STARTED=0
for s in "${STEPS[@]}"; do
    if [[ -n "${FROM}" && "${STARTED}" == "0" ]]; then
        [[ "${s}" == "${FROM}" ]] && STARTED=1 || continue
    fi
    if is_skipped "${s}"; then
        continue
    fi
    PLAN+=("${s}")
done

# ---------- 打印计划 ----------
print_env_summary "how-dev-iam 一键部署（编排）" \
    "总步骤数    : ${#STEPS[@]}" \
    "将执行步骤  : $(_arr_join ' ' "${PLAN[@]:-}")" \
    "起始步骤    : ${FROM:-(默认从头)}" \
    "跳过步骤    : $(_arr_join ' ' "${SKIP_LIST[@]:-}")" \
    "Kubeconfig  : ${KUBECONFIG:-~/.kube/config}  (来源：${KUBECONFIG_SRC})"

if [[ ${#PLAN[@]} -eq 0 ]]; then
    log_warn "计划为空，直接退出"
    exit 0
fi

# ---------- 顺序执行 ----------
_IDX=0
_TOTAL=${#PLAN[@]}

# 构造透传参数数组（家族写法）：
# - --kubeconfig 已 export，子脚本会自动继承 $KUBECONFIG；
#   同时以 --kubeconfig 参数透传，保证子脚本 env summary 能展示“--kubeconfig 参数”来源。
# - bash 3.2 下空数组展开会因 set -u 报错，这里用 :- 兄弟绕开。
_kubeconfig_passthrough=()
if [[ -n "${KUBECONFIG_FILE}" ]]; then
    _kubeconfig_passthrough=(--kubeconfig "${KUBECONFIG_FILE}")
fi

for s in "${PLAN[@]}"; do
    _IDX=$((_IDX + 1))
    log_step "${_IDX}" "${_TOTAL}" "$(step_title "$s")  →  $(step_script "$s")"
    # 注意：bash 3.2（macOS 自带）在 set -u 下展开空数组会报 unbound variable，
    # 这里改成显式分支，避免使用 "${args[@]:-}" 这种绕过语法。
    if [[ "${DRY_RUN}" == "1" ]]; then
        if [[ ${#_kubeconfig_passthrough[@]} -gt 0 ]]; then
            run_cmd bash "${SCRIPT_DIR}/$(step_script "$s")" "${_kubeconfig_passthrough[@]}" --dry-run
        else
            run_cmd bash "${SCRIPT_DIR}/$(step_script "$s")" --dry-run
        fi
    else
        if [[ ${#_kubeconfig_passthrough[@]} -gt 0 ]]; then
            run_cmd bash "${SCRIPT_DIR}/$(step_script "$s")" "${_kubeconfig_passthrough[@]}"
        else
            run_cmd bash "${SCRIPT_DIR}/$(step_script "$s")"
        fi
    fi
done

log_section "全流程完成"
log_ok "全部 ${_TOTAL} 步执行完毕"
