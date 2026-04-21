#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_FILE="${ROOT_DIR}/.sonarqube-local/state.json"

node "${ROOT_DIR}/scripts/quality-sonarqube-local.mjs" bootstrap >/dev/null

if [[ ! -f "${STATE_FILE}" ]]; then
  echo "[sast:sonarqube] Missing SonarQube local state file: ${STATE_FILE}" >&2
  exit 1
fi

SONAR_TOKEN="$(
  node -e 'const fs = require("fs"); const payload = JSON.parse(fs.readFileSync(process.argv[1], "utf8")); process.stdout.write(payload.token || "");' "${STATE_FILE}"
)"
SONAR_HOST_URL="$(
  node -e 'const fs = require("fs"); const payload = JSON.parse(fs.readFileSync(process.argv[1], "utf8")); process.stdout.write((payload.hostUrl || "").replace("localhost", "host.docker.internal"));' "${STATE_FILE}"
)"
SCANNER_IMAGE="$(
  node -e 'const fs = require("fs"); const payload = JSON.parse(fs.readFileSync(process.argv[1], "utf8")); process.stdout.write(payload.scannerImage || "sonarsource/sonar-scanner-cli:12.1.0.3225_8.0.1");' "${STATE_FILE}"
)"

if [[ -z "${SONAR_TOKEN}" || -z "${SONAR_HOST_URL}" ]]; then
  echo "[sast:sonarqube] Incomplete SonarQube state in ${STATE_FILE}" >&2
  exit 1
fi

QUALITY_ARGS=()
if [[ "${SAST_STRICT:-0}" == "1" ]]; then
  QUALITY_ARGS+=(
    "-Dsonar.qualitygate.wait=true"
    "-Dsonar.qualitygate.timeout=300"
  )
fi

docker run --rm \
  -e SONAR_HOST_URL="${SONAR_HOST_URL}" \
  -e SONAR_TOKEN="${SONAR_TOKEN}" \
  -v "${ROOT_DIR}:/usr/src" \
  -w /usr/src \
  "${SCANNER_IMAGE}" \
  -Dsonar.host.url="${SONAR_HOST_URL}" \
  -Dsonar.token="${SONAR_TOKEN}" \
  "${QUALITY_ARGS[@]}"
