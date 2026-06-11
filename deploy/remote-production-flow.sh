#!/usr/bin/env bash
set -euo pipefail

MODE=${1:-${MODE:-full}}

TEST_SERVICE=${TEST_SERVICE:-sub2api-test.service}
PRODUCTION_SERVICE=${PRODUCTION_SERVICE:-sub2api.service}
TEST_PORT=${TEST_PORT:-18808}
PRODUCTION_PORT=${PRODUCTION_PORT:-8808}
TEST_BINARY_PATH=${TEST_BINARY_PATH:-/opt/sub2api-test/sub2api}
PRODUCTION_BINARY_PATH=${PRODUCTION_BINARY_PATH:-/opt/sub2api/sub2api}
TEST_CONFIG_PATH=${TEST_CONFIG_PATH:-/opt/sub2api-test/data/config.yaml}
PRODUCTION_CONFIG_PATH=${PRODUCTION_CONFIG_PATH:-/app/data/config.yaml}
STAGE_DIR=${STAGE_DIR:-/opt/sub2api-rollout}
BINARY_NAME=${BINARY_NAME:-sub2api-linux-amd64}
PACKAGE_PATH=${PACKAGE_PATH:-}
EXPECTED_BINARY_SHA256=${EXPECTED_BINARY_SHA256:-}
EXPECTED_PACKAGE_SHA256=${EXPECTED_PACKAGE_SHA256:-}
STARTUP_WAIT_SECONDS=${STARTUP_WAIT_SECONDS:-3}
LOG_TAIL_LINES=${LOG_TAIL_LINES:-80}
KEEP_TEST_SERVICE=${KEEP_TEST_SERVICE:-0}

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
WORK_DIR=""

log() {
  printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"
}

fail() {
  log "ERROR: $*"
  exit 1
}

cleanup() {
  if [ -n "${WORK_DIR:-}" ] && [ -d "$WORK_DIR" ]; then
    rm -rf "$WORK_DIR"
  fi
}

trap cleanup EXIT

bool_true() {
  case "${1,,}" in
    1|true|yes|on) return 0 ;;
    *) return 1 ;;
  esac
}

require_cmd() {
  local cmd="$1"

  command -v "$cmd" >/dev/null 2>&1 || fail "Missing required command: $cmd"
}

require_file() {
  local path="$1"

  [ -f "$path" ] || fail "Missing file: $path"
}

sha256_of() {
  sha256sum "$1" | awk '{print $1}'
}

normalize_scalar() {
  local value="$1"

  value="$(printf '%s' "$value" | sed -E 's/[[:space:]]+#.*$//; s/^[[:space:]]+//; s/[[:space:]]+$//')"
  if [ "${#value}" -ge 2 ]; then
    if [[ "$value" == \"*\" && "$value" == *\" ]]; then
      value="${value:1:${#value}-2}"
    elif [[ "$value" == \'*\' && "$value" == *\' ]]; then
      value="${value:1:${#value}-2}"
    fi
  fi

  printf '%s' "$value"
}

yaml_get() {
  local file="$1"
  shift

  awk -v joined_path="$*" '
    function trim(s) {
      sub(/^[[:space:]]+/, "", s)
      sub(/[[:space:]]+$/, "", s)
      return s
    }
    BEGIN {
      wanted_depth = split(joined_path, wanted, " ")
    }
    /^[[:space:]]*#/ || /^[[:space:]]*$/ { next }
    {
      line = $0
      sub(/[[:space:]]+#.*/, "", line)
      indent = match(line, /[^ ]/) - 1
      if (indent < 0) next
      depth = int(indent / 2) + 1
      stripped = line
      sub(/^[[:space:]]*/, "", stripped)
      key = stripped
      sub(/:.*/, "", key)
      path[depth] = key
      for (i = depth + 1; i <= 16; i++) delete path[i]
      value = stripped
      sub(/^[^:]+:[[:space:]]*/, "", value)
      if (value == stripped) value = ""
      matched = 1
      for (i = 1; i <= wanted_depth; i++) {
        if (path[i] != wanted[i]) {
          matched = 0
          break
        }
      }
      if (matched && depth == wanted_depth) {
        print trim(value)
        exit
      }
    }
  ' "$file"
}

check_config_file() {
  local file="$1"
  local label="$2"
  local totp_key=""
  local allowlist_enabled=""
  local allow_private_hosts=""

  if [ ! -f "$file" ]; then
    log "Skipping config precheck for ${label}: ${file} not found"
    return
  fi

  totp_key="$(normalize_scalar "$(yaml_get "$file" totp encryption_key || true)")"
  if [ -z "$totp_key" ]; then
    fail "${label} config ${file} has empty totp.encryption_key"
  fi

  allowlist_enabled="$(normalize_scalar "$(yaml_get "$file" security url_allowlist enabled || true)")"
  if [ -n "$allowlist_enabled" ] && [ "$allowlist_enabled" != "true" ]; then
    fail "${label} config ${file} requires security.url_allowlist.enabled=true when that section exists"
  fi

  allow_private_hosts="$(normalize_scalar "$(yaml_get "$file" security url_allowlist allow_private_hosts || true)")"
  if [ -n "$allow_private_hosts" ] && [ "$allow_private_hosts" != "false" ]; then
    fail "${label} config ${file} requires security.url_allowlist.allow_private_hosts=false when that section exists"
  fi

  if grep -En '^[[:space:]]*[^#]+:[[:space:]]*"*http://[^"]+' "$file" | grep -Ev 'http://(127\.0\.0\.1|localhost|::1)([:/]|$)' >/dev/null 2>&1; then
    log "WARNING: ${label} config ${file} contains non-local http:// URLs. Review them before rollout."
  fi

  log "Config precheck passed: ${label} (${file})"
}

verify_service_logs() {
  local service="$1"
  local label="$2"
  local logs

  logs="$(journalctl -u "$service" -n "$LOG_TAIL_LINES" --no-pager || true)"
  printf '%s\n' "$logs"

  if printf '%s\n' "$logs" | grep -q "TOTP encryption key auto-generated"; then
    fail "${label} logs indicate TOTP encryption key was auto-generated"
  fi
}

smoke_check() {
  local label="$1"
  local service="$2"
  local binary_path="$3"
  local port="$4"
  local expected_hash="$5"
  local installed_hash=""
  local setup_body=""
  local status_body=""

  log "Verifying ${label} service state"
  systemctl is-active "$service" >/dev/null || fail "${label} service is not active: ${service}"
  status_body="$(systemctl status "$service" --no-pager || true)"
  printf '%s\n' "$status_body" | head -20

  require_file "$binary_path"
  installed_hash="$(sha256_of "$binary_path")"
  log "${label} binary SHA256: ${installed_hash}"
  if [ -n "$expected_hash" ] && [ "$installed_hash" != "$expected_hash" ]; then
    fail "${label} binary hash mismatch: expected ${expected_hash}, got ${installed_hash}"
  fi

  "$binary_path" -version
  curl -fsS "http://127.0.0.1:${port}/health" >/dev/null || fail "${label} /health check failed"
  setup_body="$(curl -fsS "http://127.0.0.1:${port}/setup/status")" || fail "${label} /setup/status check failed"
  printf '%s\n' "$setup_body" | grep -Eq '"needs_setup"[[:space:]]*:[[:space:]]*false' || fail "${label} /setup/status still reports needs_setup=true"
  curl -fsS "http://127.0.0.1:${port}/api/v1/public/pricing" >/dev/null || fail "${label} /api/v1/public/pricing check failed"
  curl -fsS "http://127.0.0.1:${port}/api/v1/monitoring/summary" >/dev/null || fail "${label} /api/v1/monitoring/summary check failed"
  verify_service_logs "$service" "$label"
}

resolve_package_path() {
  if [ -n "$PACKAGE_PATH" ]; then
    printf '%s\n' "$PACKAGE_PATH"
    return
  fi

  if [ -f "${STAGE_DIR%/}/${BINARY_NAME}.tar.zst" ]; then
    printf '%s\n' "${STAGE_DIR%/}/${BINARY_NAME}.tar.zst"
    return
  fi

  if [ -f "${STAGE_DIR%/}/${BINARY_NAME}.tar.gz" ]; then
    printf '%s\n' "${STAGE_DIR%/}/${BINARY_NAME}.tar.gz"
    return
  fi

  fail "Could not find rollout package in ${STAGE_DIR}"
}

verify_hash() {
  local label="$1"
  local file="$2"
  local expected="$3"
  local actual=""

  require_file "$file"
  actual="$(sha256_of "$file")"
  log "${label} SHA256: ${actual}"
  if [ -n "$expected" ] && [ "$actual" != "$expected" ]; then
    fail "${label} hash mismatch: expected ${expected}, got ${actual}"
  fi
}

extract_package() {
  local package="$1"

  require_file "$package"
  WORK_DIR="$(mktemp -d "${STAGE_DIR%/}/extract.XXXXXX")"

  case "$package" in
    *.tar.zst)
      if tar --help 2>/dev/null | grep -q -- '--zstd'; then
        tar --zstd -xf "$package" -C "$WORK_DIR"
      elif command -v zstd >/dev/null 2>&1; then
        zstd -dc "$package" | tar -xf - -C "$WORK_DIR"
      else
        fail "Cannot extract ${package}: tar lacks --zstd and zstd is not installed"
      fi
      ;;
    *.tar.gz|*.tgz)
      tar -xzf "$package" -C "$WORK_DIR"
      ;;
    *)
      fail "Unsupported package format: ${package}"
      ;;
  esac

  require_file "$WORK_DIR/$BINARY_NAME"
  chmod +x "$WORK_DIR/$BINARY_NAME"
  printf '%s\n' "$WORK_DIR/$BINARY_NAME"
}

backup_file() {
  local path="$1"

  if [ -f "$path" ]; then
    cp "$path" "${path}.backup.${TIMESTAMP}"
    log "Backed up ${path} -> ${path}.backup.${TIMESTAMP}"
  fi
}

install_candidate_to_test() {
  local candidate="$1"

  log "Installing rollout candidate into ${TEST_BINARY_PATH}"
  mkdir -p "$(dirname "$TEST_BINARY_PATH")"
  systemctl stop "$TEST_SERVICE" || true
  systemctl reset-failed "$TEST_SERVICE" >/dev/null 2>&1 || true
  backup_file "$TEST_BINARY_PATH"
  cp "$candidate" "$TEST_BINARY_PATH"
  chmod +x "$TEST_BINARY_PATH"
  systemctl start "$TEST_SERVICE"
  sleep "$STARTUP_WAIT_SECONDS"
}

promote_test_candidate() {
  log "Promoting verified test binary into ${PRODUCTION_BINARY_PATH}"
  require_file "$TEST_BINARY_PATH"
  mkdir -p "$(dirname "$PRODUCTION_BINARY_PATH")"
  systemctl stop "$PRODUCTION_SERVICE" || true
  systemctl reset-failed "$PRODUCTION_SERVICE" >/dev/null 2>&1 || true
  backup_file "$PRODUCTION_BINARY_PATH"
  cp "$TEST_BINARY_PATH" "$PRODUCTION_BINARY_PATH"
  chmod +x "$PRODUCTION_BINARY_PATH"
  systemctl start "$PRODUCTION_SERVICE"
  sleep "$STARTUP_WAIT_SECONDS"
}

cleanup_test_service() {
  if bool_true "$KEEP_TEST_SERVICE"; then
    log "KEEP_TEST_SERVICE enabled; leaving ${TEST_SERVICE} running"
    return
  fi

  log "Stopping and disabling ${TEST_SERVICE}"
  systemctl stop "$TEST_SERVICE" || true
  systemctl disable "$TEST_SERVICE" >/dev/null 2>&1 || true
}

precheck_configs() {
  check_config_file "$TEST_CONFIG_PATH" "test"
  if [ "$PRODUCTION_CONFIG_PATH" != "$TEST_CONFIG_PATH" ]; then
    check_config_file "$PRODUCTION_CONFIG_PATH" "production"
  fi
}

run_test_flow() {
  local package=""
  local candidate=""

  package="$(resolve_package_path)"
  verify_hash "Package" "$package" "$EXPECTED_PACKAGE_SHA256"
  candidate="$(extract_package "$package")"
  verify_hash "Candidate binary" "$candidate" "$EXPECTED_BINARY_SHA256"
  "$candidate" -version
  install_candidate_to_test "$candidate"
  smoke_check "test" "$TEST_SERVICE" "$TEST_BINARY_PATH" "$TEST_PORT" "$EXPECTED_BINARY_SHA256"
}

run_promote_flow() {
  smoke_check "test" "$TEST_SERVICE" "$TEST_BINARY_PATH" "$TEST_PORT" "$EXPECTED_BINARY_SHA256"
  promote_test_candidate
  smoke_check "production" "$PRODUCTION_SERVICE" "$PRODUCTION_BINARY_PATH" "$PRODUCTION_PORT" "$EXPECTED_BINARY_SHA256"
  cleanup_test_service
}

main() {
  require_cmd awk
  require_cmd curl
  require_cmd grep
  require_cmd journalctl
  require_cmd sha256sum
  require_cmd systemctl
  require_cmd tar

  [ "$(id -u)" -eq 0 ] || fail "Run this script as root"
  mkdir -p "$STAGE_DIR"

  case "$MODE" in
    test)
      precheck_configs
      run_test_flow
      ;;
    promote)
      precheck_configs
      run_promote_flow
      ;;
    full)
      precheck_configs
      run_test_flow
      run_promote_flow
      ;;
    *)
      fail "Unsupported mode: ${MODE}. Use test, promote, or full"
      ;;
  esac

  log "Rollout flow completed: ${MODE}"
}

main "$@"
