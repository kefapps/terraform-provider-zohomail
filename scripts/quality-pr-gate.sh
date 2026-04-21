#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

ensure_clean_worktree() {
  if [[ -n "$(git status --porcelain --untracked-files=normal)" ]]; then
    echo "[quality-pr] Refusing to certify a dirty worktree. Commit, stash, or clean local changes first."
    exit 1
  fi
}

ensure_clean_worktree

echo "[quality-pr] Building provider"
go build ./...

echo "[quality-pr] Generating Go coverage"
go test -coverprofile=coverage.out ./...

echo "[quality-pr] Starting local SonarQube"
node ./scripts/quality-sonarqube-local.mjs bootstrap >/dev/null

echo "[quality-pr] Running strict local SonarQube analysis"
SAST_STRICT=1 ./scripts/sast-sonarqube.sh
