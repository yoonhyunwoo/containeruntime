#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  exec sudo -E "$0" "$@"
fi

RUNTIME="${RUNTIME_BIN:-./containeruntime}"
BUNDLE_DIR="${BUNDLE_DIR:-/root/testbundle}"
CONTAINER_ID="stress-$(date +%s)"

if [[ ! -x "${RUNTIME}" ]]; then
  echo "runtime binary not found: ${RUNTIME}" >&2
  exit 1
fi

if [[ ! -f "${BUNDLE_DIR}/config-stress.json" ]]; then
  echo "stress config not found: ${BUNDLE_DIR}/config-stress.json" >&2
  echo "run 'make setup-stress' first" >&2
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

cat > "${BUNDLE_DIR}/config.json" <<'JSON'
{
  "ociVersion": "1.0.2",
  "process": {
    "terminal": false,
    "user": { "uid": 0, "gid": 0 },
    "args": ["/bin/dd", "if=/dev/zero", "of=/dev/null", "bs=1M", "count=256"],
    "env": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    ],
    "cwd": "/",
    "noNewPrivileges": true
  },
  "root": { "path": "/root/testbundle/stressfs", "readonly": false },
  "hostname": "containeruntime-stress",
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
"${RUNTIME}" start "${CONTAINER_ID}"
sleep 3

state_after="$(${RUNTIME} state ${CONTAINER_ID})"
printf '%s\n' "${state_after}" | grep -Eq '"status": "(running|stopped)"'

delete_with_retry

echo "stress test passed: ${CONTAINER_ID}"
