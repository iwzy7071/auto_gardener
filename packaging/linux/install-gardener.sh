#!/usr/bin/env bash
set -euo pipefail

RELAY_BASE_URL="${GARDENER_RELAY_BASE_URL:-}"
INSTALL_DIR="$HOME/.local/share/Gardener"
SETUP_KEY=""
PROVISION_URL=""
START_AFTER_INSTALL=1
PACKAGE_SHA256_URL=""
PROVISION_JSON_MAX_BYTES=$((64 * 1024))

usage() { echo "Usage: install-gardener.sh [--setup-key sk_xxx] [--relay-base-url URL] [--install-dir DIR] [--no-start]"; }
validate_install_dir() {
  local dir="${1:-}"
  case "$dir" in ""|"/"|"."|".."|../*|*/../*) echo "Refusing unsafe install directory: $dir" >&2; exit 1 ;; esac
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --setup-key|-k) SETUP_KEY="${2:-}"; shift 2 ;;
    --provision-url) PROVISION_URL="${2:-}"; shift 2 ;;
    --relay-base-url) RELAY_BASE_URL="${2:-}"; shift 2 ;;
    --install-dir) INSTALL_DIR="${2:-}"; shift 2 ;;
    --package-sha256-url) PACKAGE_SHA256_URL="${2:-}"; shift 2 ;;
    --start|--start-after-install) START_AFTER_INSTALL=1; shift ;;
    --no-start) START_AFTER_INSTALL=0; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

validate_install_dir "$INSTALL_DIR"
RELAY_BASE_URL="${RELAY_BASE_URL%/}"
if [[ -n "$SETUP_KEY" && ! "$SETUP_KEY" =~ ^[A-Za-z0-9_-]{20,}$ ]]; then echo "Setup key format is invalid." >&2; exit 1; fi
is_placeholder_relay_url() { [[ -z "${1:-}" || "$1" == *YOUR_RELAY_SERVER* || "$1" == *YOUR_SERVER_IP* || "$1" == *example.com* ]]; }
if [[ -z "$PROVISION_URL" && -n "$SETUP_KEY" ]]; then
  if is_placeholder_relay_url "$RELAY_BASE_URL"; then echo "Relay base URL is not configured. Re-run with --relay-base-url http://YOUR_SERVER or set GARDENER_RELAY_BASE_URL." >&2; exit 1; fi
  PROVISION_URL="$RELAY_BASE_URL/downloads/provision/$SETUP_KEY/gardener.provision.json"
fi

case "$(uname -m)" in
  x86_64|amd64) pkg_arch="amd64" ;;
  aarch64|arm64) pkg_arch="arm64" ;;
  *) echo "Unsupported Linux architecture: $(uname -m)" >&2; exit 1 ;;
esac
PACKAGE_URL="${RELAY_BASE_URL:+$RELAY_BASE_URL/downloads/Gardener-Linux-$pkg_arch.tar.gz}"
DATA_DIR="$HOME/forest_data"; [[ -d "$HOME/Desktop" ]] && DATA_DIR="$HOME/Desktop/forest_data"
BASE_INSTALL_PATH="$PATH"
INSTALL_PATH="$BASE_INSTALL_PATH:/usr/local/bin:/usr/bin:/bin:$HOME/.local/bin:$HOME/.npm-global/bin"
CODEX_CMD="$(PATH="$INSTALL_PATH" command -v codex || true)"
CLAUDE_CMD="$(PATH="$INSTALL_PATH" command -v claude || true)"
TMP="$(mktemp -d)"; cleanup(){ rm -rf "$TMP"; }; trap cleanup EXIT
shell_quote() { python3 -c 'import shlex, sys; print(shlex.quote(sys.argv[1]), end="")' "$1"; }

safe_extract_tar() {
  python3 - "$1" "$2" <<'PYINNER'
import pathlib, sys, tarfile
archive, dest = sys.argv[1:3]
root = pathlib.Path(dest).resolve()
with tarfile.open(archive, 'r:gz') as tf:
    for member in tf.getmembers():
        name = member.name
        if name.startswith('/') or name == '..' or name.startswith('../') or '/../' in name:
            raise SystemExit(f'Unsafe archive path: {name}')
        target = (root / name).resolve()
        if target != root and root not in target.parents:
            raise SystemExit(f'Unsafe archive path: {name}')
        if member.issym() or member.islnk():
            raise SystemExit(f'Archive links are not allowed: {name}')
    tf.extractall(root)
PYINNER
}

verify_sha256_file() {
  local file="$1" sha_url="$2" sha_file="$3"
  [[ -z "$sha_url" ]] && return 0
  echo "Verifying Gardener package checksum..."
  curl -fsSL --connect-timeout 20 --max-time 120 "$sha_url" -o "$sha_file"
  local expected
  expected="$(python3 - "$sha_file" <<'PYINNER'
import re, sys
text=open(sys.argv[1], encoding='utf-8', errors='ignore').read()
m=re.search(r'(?i)\b[0-9a-f]{64}\b', text)
if not m: raise SystemExit('missing sha256 digest')
print(m.group(0).lower())
PYINNER
)"
  printf '%s  %s\n' "$expected" "$file" | sha256sum -c - >/dev/null
}

PROVISION_JSON="$TMP/provision.json"
if [[ -n "$PROVISION_URL" ]]; then
  echo "Loading Gardener relay provision..."
  curl -fsSL --connect-timeout 20 --max-time 120 "$PROVISION_URL" -o "$PROVISION_JSON"
  provision_size="$(wc -c < "$PROVISION_JSON" | tr -d '[:space:]')"
  [[ "$provision_size" -gt "$PROVISION_JSON_MAX_BYTES" ]] && { echo "Relay provision JSON is too large." >&2; exit 1; }
fi

if [[ -s "$PROVISION_JSON" ]]; then
  provision_package_info="$(python3 - "$PROVISION_JSON" "$pkg_arch" <<'PY'
import json, sys
j=json.load(open(sys.argv[1])); arch=sys.argv[2]
urls=j.get('linuxPackageUrls') or {}; sha_urls=j.get('linuxPackageSha256Urls') or {}
print(urls.get(arch) or j.get('linuxPackageUrl') or '')
print(sha_urls.get(arch) or j.get('linuxPackageSha256Url') or j.get('packageSha256Url') or '')
PY
)"
  package_from_provision="$(printf '%s\n' "$provision_package_info" | sed -n '1p')"
  package_sha_from_provision="$(printf '%s\n' "$provision_package_info" | sed -n '2p')"
  [[ -n "$package_from_provision" ]] && PACKAGE_URL="$package_from_provision"
  [[ -n "$package_sha_from_provision" ]] && PACKAGE_SHA256_URL="$package_sha_from_provision"
fi
[[ -z "$PACKAGE_URL" ]] && { echo "Package URL is not configured. Re-run with --relay-base-url http://YOUR_SERVER or set GARDENER_RELAY_BASE_URL." >&2; exit 1; }

mkdir -p "$INSTALL_DIR"
echo "Downloading Gardener package..."
curl -fL --connect-timeout 20 --max-time 300 "$PACKAGE_URL" -o "$TMP/gardener.tar.gz"
verify_sha256_file "$TMP/gardener.tar.gz" "$PACKAGE_SHA256_URL" "$TMP/gardener.tar.gz.sha256"
mkdir -p "$TMP/extract"; safe_extract_tar "$TMP/gardener.tar.gz" "$TMP/extract"
SRC="$(find "$TMP/extract" -maxdepth 1 -type d -name 'Gardener-Linux-*' | head -n 1)"; [[ -z "$SRC" ]] && SRC="$TMP/extract"

command -v systemctl >/dev/null 2>&1 && systemctl --user stop gardener.relay.service gardener.local.service >/dev/null 2>&1 || true
backup="$INSTALL_DIR/backup-$(date +%Y%m%d-%H%M%S)"; mkdir -p "$backup"
for name in gardener frpc web install-gardener.sh start-gardener.sh update-gardener.sh README-Linux.txt; do [[ -e "$INSTALL_DIR/$name" ]] && mv "$INSTALL_DIR/$name" "$backup/$name"; done
for name in gardener frpc web install-gardener.sh start-gardener.sh update-gardener.sh README-Linux.txt; do [[ -e "$SRC/$name" ]] && cp -R "$SRC/$name" "$INSTALL_DIR/$name"; done
chmod +x "$INSTALL_DIR/gardener" "$INSTALL_DIR/start-gardener.sh" "$INSTALL_DIR/update-gardener.sh" "$INSTALL_DIR/install-gardener.sh" 2>/dev/null || true
[[ -f "$INSTALL_DIR/frpc" ]] && chmod +x "$INSTALL_DIR/frpc" 2>/dev/null || true

if [[ -s "$PROVISION_JSON" ]]; then
  echo "Writing relay configuration..."
  python3 - "$PROVISION_JSON" "$INSTALL_DIR" "$DATA_DIR" <<'PY'
import json, pathlib, sys, datetime
provision_path, install_dir, data_dir = sys.argv[1:4]
j=json.load(open(provision_path)); root=pathlib.Path(install_dir)
if j.get('frpcToml'):
    frpc_path=root/'frpc.toml'; frpc_path.write_text(j['frpcToml'], encoding='utf-8'); frpc_path.chmod(0o600)
relay={'schemaVersion':1,'user':j.get('user',''),'publicUrl':j.get('publicUrl',''),'webUsername':j.get('webUsername',''),'webPassword':j.get('webPassword',''),'installedAt':datetime.datetime.now(datetime.timezone.utc).isoformat()}
relay_path=root/'gardener.relay.json'; relay_path.write_text(json.dumps(relay, ensure_ascii=False, indent=2)+'\n', encoding='utf-8'); relay_path.chmod(0o600)
tokens=j.get('providerTokens') or {}; minimax=str(tokens.get('minimaxToken') or '').strip(); kimi=str(tokens.get('kimiToken') or '').strip()
if minimax or kimi:
    data=pathlib.Path(data_dir); data.mkdir(parents=True, exist_ok=True); settings_path=data/'settings.json'
    settings={'logLevel':'quiet','modelMode':'default','cliEngine':'codex'}
    if settings_path.exists():
        try:
            existing=json.loads(settings_path.read_text(encoding='utf-8'))
            if isinstance(existing, dict): settings.update(existing)
        except Exception: pass
    if minimax and not str(settings.get('minimaxToken') or '').strip(): settings['minimaxToken']=minimax
    if kimi and not str(settings.get('kimiToken') or '').strip(): settings['kimiToken']=kimi
    settings_path.write_text(json.dumps(settings, ensure_ascii=False, indent=2)+'\n', encoding='utf-8'); settings_path.chmod(0o600)
PY
fi

cat > "$INSTALL_DIR/gardener.config.sh" <<CONFIG_EOF
# Generated by install-gardener.sh. Used by shell fallback scripts.
export PATH=$(shell_quote "$INSTALL_PATH")
export HOME=$(shell_quote "$HOME")
export USER=$(shell_quote "$(whoami)")
export AUTO_GARDENER_ADDR=$(shell_quote "127.0.0.1:8080")
export AUTO_GARDENER_STATIC=$(shell_quote "$INSTALL_DIR/web/static")
export AUTO_GARDENER_DATA=$(shell_quote "$DATA_DIR")
CONFIG_EOF
[[ -n "$CODEX_CMD" ]] && printf 'export AUTO_GARDENER_CODEX_CMD=%s\n' "$(shell_quote "$CODEX_CMD")" >> "$INSTALL_DIR/gardener.config.sh"
[[ -n "$CLAUDE_CMD" ]] && printf 'export AUTO_GARDENER_CLAUDE_CMD=%s\n' "$(shell_quote "$CLAUDE_CMD")" >> "$INSTALL_DIR/gardener.config.sh"
cat > "$INSTALL_DIR/gardener.env" <<ENV_EOF
# Generated by install-gardener.sh. Used by systemd user services.
PATH=$INSTALL_PATH
HOME=$HOME
USER=$(whoami)
AUTO_GARDENER_ADDR=127.0.0.1:8080
AUTO_GARDENER_STATIC=$INSTALL_DIR/web/static
AUTO_GARDENER_DATA=$DATA_DIR
ENV_EOF
[[ -n "$CODEX_CMD" ]] && printf 'AUTO_GARDENER_CODEX_CMD=%s\n' "$CODEX_CMD" >> "$INSTALL_DIR/gardener.env"
[[ -n "$CLAUDE_CMD" ]] && printf 'AUTO_GARDENER_CLAUDE_CMD=%s\n' "$CLAUDE_CMD" >> "$INSTALL_DIR/gardener.env"
chmod 600 "$INSTALL_DIR/gardener.config.sh" "$INSTALL_DIR/gardener.env" 2>/dev/null || true
mkdir -p "$INSTALL_DIR/logs" "$HOME/.config/systemd/user"

cat > "$HOME/.config/systemd/user/gardener.local.service" <<SERVICE_EOF
[Unit]
Description=Gardener local web service
After=network-online.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$INSTALL_DIR/gardener.env
ExecStart=$INSTALL_DIR/gardener
Restart=always
RestartSec=3
StandardOutput=append:$INSTALL_DIR/logs/gardener.local.out.log
StandardError=append:$INSTALL_DIR/logs/gardener.local.err.log

[Install]
WantedBy=default.target
SERVICE_EOF

if [[ -f "$INSTALL_DIR/frpc.toml" && -x "$INSTALL_DIR/frpc" ]]; then
cat > "$HOME/.config/systemd/user/gardener.relay.service" <<SERVICE_EOF
[Unit]
Description=Gardener relay tunnel client
After=network-online.target gardener.local.service

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/frpc -c $INSTALL_DIR/frpc.toml
Restart=always
RestartSec=3
StandardOutput=append:$INSTALL_DIR/logs/gardener.relay.out.log
StandardError=append:$INSTALL_DIR/logs/gardener.relay.err.log

[Install]
WantedBy=default.target
SERVICE_EOF
fi

if [[ "$START_AFTER_INSTALL" == "1" ]]; then
  if command -v systemctl >/dev/null 2>&1; then
    systemctl --user daemon-reload >/dev/null 2>&1 || true
    systemctl --user enable --now gardener.local.service >/dev/null 2>&1 || true
    [[ -f "$HOME/.config/systemd/user/gardener.relay.service" ]] && systemctl --user enable --now gardener.relay.service >/dev/null 2>&1 || true
  fi
  "$INSTALL_DIR/start-gardener.sh" >/dev/null 2>&1 || true
fi

local_url="http://127.0.0.1:8080"; public_url=""; web_user=""
if [[ -f "$INSTALL_DIR/gardener.relay.json" ]]; then
  public_url="$(python3 - "$INSTALL_DIR/gardener.relay.json" <<'PY'
import json,sys
j=json.load(open(sys.argv[1])); print(j.get('publicUrl',''))
PY
)"
  web_user="$(python3 - "$INSTALL_DIR/gardener.relay.json" <<'PY'
import json,sys
j=json.load(open(sys.argv[1])); print(j.get('webUsername',''))
PY
)"
fi

echo "Gardener installed."
echo "Local URL:  $local_url"
if [[ "$public_url" == http://* || "$public_url" == https://* ]]; then
  echo "Remote URL: $public_url"
  echo "Login user: $web_user"
  echo "Login password: saved in gardener.relay.json; keep that file private."
fi
[[ "$START_AFTER_INSTALL" == "1" ]] && command -v xdg-open >/dev/null 2>&1 && xdg-open "${public_url:-$local_url}" >/dev/null 2>&1 || true
