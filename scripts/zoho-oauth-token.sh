#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/zoho-oauth-token.sh [--env-file PATH] [--dry-run]

Loads Zoho OAuth bootstrap values from an env file, then:
- exchanges ZOHOMAIL_AUTHORIZATION_CODE for access + refresh tokens when no refresh token exists
- otherwise refreshes ZOHOMAIL_ACCESS_TOKEN from ZOHOMAIL_REFRESH_TOKEN

The env file is updated in place.
EOF
}

env_file=".env.testacc"
dry_run=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-file)
      [[ $# -ge 2 ]] || {
        echo "--env-file requires a path" >&2
        exit 1
      }
      env_file="$2"
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ ! -f "$env_file" ]]; then
  echo "env file not found: $env_file" >&2
  exit 1
fi

while IFS= read -r raw_line || [[ -n "$raw_line" ]]; do
  line="${raw_line#"${raw_line%%[![:space:]]*}"}"
  if [[ -z "$line" || "${line:0:1}" == "#" || "$line" != *=* ]]; then
    continue
  fi

  key="${line%%=*}"
  key="$(printf '%s' "$key" | awk '{$1=$1; print}')"
  value="${line#*=}"

  if [[ "$value" =~ ^\".*\"$ || "$value" =~ ^\'.*\'$ ]]; then
    value="${value:1:${#value}-2}"
  fi

  export "$key=$value"
done <"$env_file"

trim() {
  printf '%s' "$1" | awk '{$1=$1; print}'
}

require_var() {
  local name="$1"
  local value
  value="$(trim "${!name:-}")"
  if [[ -z "$value" ]]; then
    echo "missing required env var: $name" >&2
    exit 1
  fi
}

accounts_url_for_dc() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    us) printf '%s\n' 'https://accounts.zoho.com' ;;
    eu) printf '%s\n' 'https://accounts.zoho.eu' ;;
    in) printf '%s\n' 'https://accounts.zoho.in' ;;
    au) printf '%s\n' 'https://accounts.zoho.com.au' ;;
    jp) printf '%s\n' 'https://accounts.zoho.jp' ;;
    ca) printf '%s\n' 'https://accounts.zohocloud.ca' ;;
    cn) printf '%s\n' 'https://accounts.zoho.com.cn' ;;
    ae) printf '%s\n' 'https://accounts.zoho.ae' ;;
    sa) printf '%s\n' 'https://accounts.zoho.sa' ;;
    *)
      echo "unsupported ZOHOMAIL_DATA_CENTER: $1" >&2
      echo "set ZOHOMAIL_ACCOUNTS_URL in the env file if you need a manual override" >&2
      exit 1
      ;;
  esac
}

require_var "ZOHOMAIL_DATA_CENTER"
require_var "ZOHOMAIL_CLIENT_ID"
require_var "ZOHOMAIL_CLIENT_SECRET"

zoho_accounts_url="$(trim "${ZOHOMAIL_ACCOUNTS_URL:-}")"
if [[ -z "$zoho_accounts_url" ]]; then
  zoho_accounts_url="$(accounts_url_for_dc "$(trim "${ZOHOMAIL_DATA_CENTER}")")"
fi

grant_type=""
if [[ -n "$(trim "${ZOHOMAIL_REFRESH_TOKEN:-}")" ]]; then
  grant_type="refresh_token"
elif [[ -n "$(trim "${ZOHOMAIL_AUTHORIZATION_CODE:-}")" ]]; then
  grant_type="authorization_code"
else
  echo "set ZOHOMAIL_REFRESH_TOKEN or ZOHOMAIL_AUTHORIZATION_CODE in $env_file" >&2
  exit 1
fi

if [[ "$dry_run" -eq 1 ]]; then
  echo "env_file=$env_file"
  echo "accounts_url=$zoho_accounts_url"
  echo "grant_type=$grant_type"
  exit 0
fi

response_file="$(mktemp "${TMPDIR:-/tmp}/zoho-oauth-response.XXXXXX")"
trap 'rm -f "$response_file"' EXIT

curl_args=(
  -fsS
  -X POST
  "${zoho_accounts_url}/oauth/v2/token"
  --data-urlencode "client_id=${ZOHOMAIL_CLIENT_ID}"
  --data-urlencode "client_secret=${ZOHOMAIL_CLIENT_SECRET}"
  --data-urlencode "grant_type=${grant_type}"
)

if [[ "$grant_type" == "refresh_token" ]]; then
  curl_args+=(--data-urlencode "refresh_token=${ZOHOMAIL_REFRESH_TOKEN}")
else
  curl_args+=(--data-urlencode "code=${ZOHOMAIL_AUTHORIZATION_CODE}")
fi

curl "${curl_args[@]}" >"$response_file"

parsed_json="$(
  python3 - "$response_file" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
payload = json.loads(path.read_text())

if payload.get("error"):
    details = payload.get("error_description") or payload["error"]
    raise SystemExit(f"zoho oauth error: {details}")

access_token = payload.get("access_token", "")
if not access_token:
    raise SystemExit("zoho oauth response did not contain access_token")

print(json.dumps({
    "access_token": access_token,
    "refresh_token": payload.get("refresh_token", ""),
    "expires_in": str(payload.get("expires_in", "")),
}))
PY
)"

export ZOHO_OAUTH_RESULT_JSON="$parsed_json"
export ZOHO_OAUTH_ENV_FILE="$env_file"
export ZOHO_OAUTH_GRANT_TYPE="$grant_type"

python3 <<'PY'
import json
import os
import pathlib
import shlex

env_path = pathlib.Path(os.environ["ZOHO_OAUTH_ENV_FILE"])
result = json.loads(os.environ["ZOHO_OAUTH_RESULT_JSON"])
grant_type = os.environ["ZOHO_OAUTH_GRANT_TYPE"]

updates = {
    "ZOHOMAIL_ACCESS_TOKEN": result["access_token"],
}

refresh_token = result.get("refresh_token", "")
if refresh_token:
    updates["ZOHOMAIL_REFRESH_TOKEN"] = refresh_token

if grant_type == "authorization_code":
    updates["ZOHOMAIL_AUTHORIZATION_CODE"] = ""

lines = env_path.read_text().splitlines()
written = set()
output = []

for line in lines:
    stripped = line.strip()
    if not stripped or stripped.startswith("#") or "=" not in line:
      output.append(line)
      continue

    key, _ = line.split("=", 1)
    key = key.strip()

    if key not in updates:
      output.append(line)
      continue

    value = updates[key]
    if value == "":
      output.append(f"{key}=")
    else:
      output.append(f"{key}={shlex.quote(value)}")
    written.add(key)

for key, value in updates.items():
    if key in written:
      continue
    if value == "":
      output.append(f"{key}=")
    else:
      output.append(f"{key}={shlex.quote(value)}")

env_path.write_text("\n".join(output) + "\n")
PY

echo "updated $env_file"
echo "grant_type=$grant_type"
echo "access_token refreshed"
if [[ "$grant_type" == "authorization_code" ]]; then
  echo "refresh_token stored"
  echo "authorization code cleared"
fi
