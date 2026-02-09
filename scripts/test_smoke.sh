#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  exec sudo -E "$0" "$@"
fi

RUNTIME="${RUNTIME_BIN:-./containeruntime}"
BUNDLE_DIR="${BUNDLE_DIR:-/root/testbundle}"
ROOTFS_PATH="${ROOTFS_PATH:-/root/testbundle/ubuntufs}"
CONTAINER_ID="smoke-$(date +%s)"
SMOKE_MARKER="SMOKE_OK_${CONTAINER_ID}"
LOG_FILE="${LOG_FILE:-/tmp/containeruntime-smoke.log}"

if [[ ! -x "${RUNTIME}" ]]; then
  echo "runtime binary not found: ${RUNTIME}" >&2
  exit 1
fi

if [[ ! -d "${ROOTFS_PATH}" ]]; then
  echo "rootfs not found: ${ROOTFS_PATH}" >&2
  echo "run 'make setup-ubuntu' first" >&2
  exit 1
fi

cleanup() {
  delete_with_retry >/dev/null 2>&1 || true
}
trap cleanup EXIT

delete_with_retry() {
  local i
  for i in 1 2 3 4 5; do
    if "${RUNTIME}" delete "${CONTAINER_ID}"; then
      return 0
    fi
    if ! "${RUNTIME}" state "${CONTAINER_ID}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

rm -f "${LOG_FILE}"
exec > >(tee -a "${LOG_FILE}") 2>&1

cat > "${BUNDLE_DIR}/config.json" <<JSON
{
  "ociVersion": "1.0.2",
  "process": {
    "terminal": false,
    "user": { "uid": 0, "gid": 0 },
    "args": ["/bin/echo", "${SMOKE_MARKER}"],
    "env": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    ],
    "cwd": "/",
    "noNewPrivileges": true
  },
  "root": { "path": "${ROOTFS_PATH}", "readonly": false },
  "hostname": "containeruntime-smoke",
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

"${RUNTIME}" create "${CONTAINER_ID}" "${BUNDLE_DIR}"
state_created="$(${RUNTIME} state ${CONTAINER_ID})"
printf '%s\n' "${state_created}" | grep -q '"status": "created"'

"${RUNTIME}" start "${CONTAINER_ID}"
sleep 1
grep -q "${SMOKE_MARKER}" "${LOG_FILE}"
state_running="$(${RUNTIME} state ${CONTAINER_ID})"
printf '%s\n' "${state_running}" | grep -Eq '"status": "(running|stopped)"'

delete_with_retry
if "${RUNTIME}" state "${CONTAINER_ID}" >/dev/null 2>&1; then
  echo "container state still exists after delete: ${CONTAINER_ID}" >&2
  exit 1
fi

echo "smoke test passed: ${CONTAINER_ID}"
