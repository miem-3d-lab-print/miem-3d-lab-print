#!/usr/bin/env bash

set -Eeuo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
readonly DOCKER_CONFIG_DIR="/etc/docker"
readonly DOCKER_CONFIG="${DOCKER_CONFIG_DIR}/daemon.json"
readonly DOCKER_IPV6_SUBNET="fd00:3d1a:b001::/64"

temporary_config=""

log() {
  printf '[deploy] %s\n' "$*"
}

fail() {
  printf '[deploy] ERROR: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [[ -n "${temporary_config}" && -f "${temporary_config}" ]]; then
    rm -f -- "${temporary_config}"
  fi
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command is not installed: $1"
}

ensure_root() {
  if [[ "${EUID}" -eq 0 ]]; then
    return
  fi

  require_command sudo
  log "requesting administrator privileges"
  exec sudo -- "$0" "$@"
}

install_jq() {
  if command -v jq >/dev/null 2>&1; then
    return
  fi

  require_command apt-get
  log "installing jq for a safe daemon.json update"
  DEBIAN_FRONTEND=noninteractive apt-get update
  DEBIAN_FRONTEND=noninteractive apt-get install -y jq
}

check_host_ipv6() {
  [[ -s /proc/net/if_inet6 ]] || fail "IPv6 is disabled on the host"

  if command -v ip >/dev/null 2>&1 && [[ -z "$(ip -6 route show default)" ]]; then
    fail "the host has no default IPv6 route; configure IPv6 at the hosting provider first"
  fi
}

configure_docker_ipv6() {
  local backup_path

  install -d -m 0755 "${DOCKER_CONFIG_DIR}"
  if [[ ! -f "${DOCKER_CONFIG}" ]]; then
    printf '{}\n' >"${DOCKER_CONFIG}"
    chmod 0644 "${DOCKER_CONFIG}"
  fi

  temporary_config="$(mktemp)"
  jq -e --arg subnet "${DOCKER_IPV6_SUBNET}" '
    if type == "object" then . else error("daemon.json must contain a JSON object") end
    |
    .ipv6 = true
    | if has("fixed-cidr-v6") then . else .["fixed-cidr-v6"] = $subnet end
    | .ip6tables = true
  ' "${DOCKER_CONFIG}" >"${temporary_config}"

  dockerd --validate --config-file="${temporary_config}" >/dev/null

  if cmp -s "${DOCKER_CONFIG}" "${temporary_config}"; then
    log "Docker Engine IPv6 configuration is already current"
    return
  fi

  backup_path="${DOCKER_CONFIG}.before-ipv6.$(date -u +%Y%m%dT%H%M%SZ)"
  cp -a -- "${DOCKER_CONFIG}" "${backup_path}"
  install -m 0644 "${temporary_config}" "${DOCKER_CONFIG}"
  log "saved the previous Docker configuration to ${backup_path}"

  if ! systemctl restart docker || ! systemctl is-active --quiet docker; then
    cp -a -- "${backup_path}" "${DOCKER_CONFIG}"
    systemctl restart docker || true
    fail "Docker rejected the IPv6 configuration; the previous daemon.json was restored"
  fi

  log "Docker Engine restarted with IPv6 enabled"
}

deploy_application() {
  cd -- "${PROJECT_DIR}"

  [[ -f .env ]] || fail "${PROJECT_DIR}/.env is missing; create it from .env.example and set production secrets"
  docker compose config --quiet

  log "recreating the application network and containers"
  docker compose down --remove-orphans

  if ! docker compose up --build -d; then
    docker compose ps || true
    docker compose logs --no-color --tail=100 migrator backend || true
    fail "Docker Compose could not start the application"
  fi

  if ! docker compose exec -T backend \
    awk '$6 != "lo" { found = 1 } END { exit !found }' /proc/net/if_inet6; then
    fail "the backend container started without IPv6"
  fi

  docker compose ps
  log "deployment completed; the public entry point is port 3000"
}

main() {
  ensure_root "$@"
  require_command docker
  require_command dockerd
  require_command systemctl
  install_jq
  docker compose version >/dev/null
  check_host_ipv6
  configure_docker_ipv6
  deploy_application
}

trap cleanup EXIT
main "$@"
