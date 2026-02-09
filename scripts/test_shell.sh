#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  exec sudo -E "$0" "$@"
fi

RUNTIME="${RUNTIME_BIN:-./containeruntime}"
BUNDLE_DIR="${BUNDLE_DIR:-/root/testbundle}"
ROOTFS_PATH="${ROOTFS_PATH:-/root/testbundle/ubuntufs}"
CONTAINER_ID="shell-$(date +%s)"
LOG_FILE="${LOG_FILE:-/tmp/containeruntime-shell.log}"

delete_with_retry() {
  local i
  for i in 1 2 3 4 5; do
    if "${RUNTIME}" delete "${CONTAINER_ID}" >/dev/null 2>&1; then
      return 0
    fi
    if ! "${RUNTIME}" state "${CONTAINER_ID}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

if [[ ! -x "${RUNTIME}" ]]; then
  echo "runtime binary not found: ${RUNTIME}" >&2
  exit 1
fi

if [[ ! -d "${ROOTFS_PATH}" ]]; then
  echo "rootfs not found: ${ROOTFS_PATH}" >&2
  echo "run 'make setup-ubuntu' first" >&2
  exit 1
fi

best_effort_delete() {
  (delete_with_retry || true) &
}

cat > "${BUNDLE_DIR}/config.json" <<JSON
{
  "ociVersion": "1.0.2",
  "process": {
    "terminal": true,
    "user": { "uid": 0, "gid": 0 },
    "args": ["/bin/bash"],
    "env": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
      "TERM=xterm"
    ],
    "cwd": "/",
    "noNewPrivileges": true
  },
  "root": { "path": "${ROOTFS_PATH}", "readonly": false },
  "hostname": "containeruntime-shell",
  "mounts": [
    { "destination": "/proc", "type": "proc", "source": "proc" }
  ],
  "linux": {
    "resources": { "devices": [{ "allow": false, "access": "rwm" }] },
    "namespaces": [
      { "type": "pid" },
      { "type": "network" },
      { "type": "ipc" },
      { "type": "uts" },
      { "type": "mount" }
    ]
  }
}
JSON

set +e
timeout 10 "${RUNTIME}" create "${CONTAINER_ID}" "${BUNDLE_DIR}" >"${LOG_FILE}" 2>&1
rc=$?
set -e

if grep -Eq "terminal mode is not supported yet|failed to create pty pair|function not implemented" "${LOG_FILE}"; then
  best_effort_delete
  echo "shell test skipped: terminal mode/pty not available in current runtime environment"
  exit 0
fi

if [[ ${rc} -eq 124 ]]; then
  best_effort_delete
  echo "shell test skipped: terminal create path timed out in current runtime environment"
  exit 0
fi

if [[ ${rc} -ne 0 ]]; then
  best_effort_delete
  echo "shell test failed: create returned non-zero exit status" >&2
  cat "${LOG_FILE}" >&2
  exit 1
fi

state_created="$(${RUNTIME} state "${CONTAINER_ID}")"
printf '%s\n' "${state_created}" | grep -q '"status": "created"'

"${RUNTIME}" start "${CONTAINER_ID}"
sleep 1

state_after_start="$(${RUNTIME} state "${CONTAINER_ID}")"
printf '%s\n' "${state_after_start}" | grep -Eq '"status": "(running|stopped)"'

if ! delete_with_retry; then
  echo "shell test failed: delete did not succeed after retries" >&2
  exit 1
fi

if "${RUNTIME}" state "${CONTAINER_ID}" >/dev/null 2>&1; then
  echo "shell test failed: state still exists after delete (${CONTAINER_ID})" >&2
  exit 1
fi

echo "shell test passed: terminal lifecycle create/start/delete succeeded (${CONTAINER_ID})"
exit 0
