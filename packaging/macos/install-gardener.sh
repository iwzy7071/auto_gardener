#!/usr/bin/env bash
set -euo pipefail

RELAY_BASE_URL="${GARDENER_RELAY_BASE_URL:-}"
INSTALL_DIR="$HOME/Applications/Gardener"
SETUP_KEY=""
PROVISION_URL=""
START_AFTER_INSTALL=1
PACKAGE_SHA256_URL=""
PROVISION_JSON_MAX_BYTES=$((64 * 1024))

validate_install_dir() {
  local dir="${1:-}"
  case "$dir" in
    ""|"/"|"."|".."|../*|*/../*)
      echo "Refusing unsafe install directory: $dir" >&2
      exit 1
      ;;
  esac
}

usage() {
  cat <<EOF
Usage: install-gardener.sh [--setup-key sk_xxx] [--relay-base-url URL] [--install-dir DIR] [--no-start]
EOF
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
if [[ -n "$SETUP_KEY" && ! "$SETUP_KEY" =~ ^[A-Za-z0-9_-]{20,}$ ]]; then
  echo "Setup key format is invalid." >&2
  exit 1
fi

is_placeholder_relay_url() { [[ -z "${1:-}" || "$1" == *YOUR_RELAY_SERVER* || "$1" == *YOUR_SERVER_IP* || "$1" == *example.com* ]]; }
if [[ -z "$PROVISION_URL" && -n "$SETUP_KEY" ]]; then
  if is_placeholder_relay_url "$RELAY_BASE_URL"; then
    echo "Relay base URL is not configured. Re-run with --relay-base-url http://YOUR_SERVER or set GARDENER_RELAY_BASE_URL." >&2
    exit 1
  fi
  PROVISION_URL="$RELAY_BASE_URL/downloads/provision/$SETUP_KEY/gardener.provision.json"
fi

arch="$(uname -m)"
case "$arch" in
  arm64) pkg_arch="arm64" ;;
  x86_64|amd64) pkg_arch="amd64" ;;
  *) echo "Unsupported macOS architecture: $arch" >&2; exit 1 ;;
esac
PACKAGE_URL="${RELAY_BASE_URL:+$RELAY_BASE_URL/downloads/Gardener-macOS-$pkg_arch.tar.gz}"

# Capture the interactive shell PATH/CLI locations at install time. LaunchAgent's
# default PATH is too small on macOS and otherwise cannot find Homebrew CLIs.
MAX_INSTALL_PATH_LEN=8192
BASE_INSTALL_PATH="$PATH"
if (( ${#BASE_INSTALL_PATH} > MAX_INSTALL_PATH_LEN )); then
  echo "Warning: PATH is too long; not inheriting the existing PATH." >&2
  BASE_INSTALL_PATH=""
fi
INSTALL_PATH="$BASE_INSTALL_PATH:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/local/sbin:$HOME/.local/bin"
CODEX_CMD="$(PATH="$INSTALL_PATH" command -v codex || true)"
CLAUDE_CMD="$(PATH="$INSTALL_PATH" command -v claude || true)"

TMP="$(mktemp -d)"
cleanup(){ rm -rf "$TMP"; }
trap cleanup EXIT


verify_sha256_file() {
  local file="$1" sha_url="$2" sha_file="$3"
  if [[ -z "$sha_url" ]]; then return 0; fi
  echo "Verifying Gardener package checksum..."
  curl -fsSL --connect-timeout 20 --max-time 120 "$sha_url" -o "$sha_file"
  local expected
  expected="$(python3 - "$sha_file" <<'PYINNER'
import re, sys
text=open(sys.argv[1], encoding='utf-8', errors='ignore').read()
m=re.search(r'(?i)\b[0-9a-f]{64}\b', text)
if not m:
    raise SystemExit('missing sha256 digest')
print(m.group(0).lower())
PYINNER
)"
  printf '%s  %s\n' "$expected" "$file" | shasum -a 256 -c - >/dev/null
}

safe_extract_tar() {
  local archive="$1" dest="$2"
  python3 - "$archive" "$dest" <<'PYINNER'
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


PROVISION_JSON="$TMP/provision.json"
if [[ -n "$PROVISION_URL" ]]; then
  echo "Loading Gardener relay provision..."
  curl -fsSL --connect-timeout 20 --max-time 120 "$PROVISION_URL" -o "$PROVISION_JSON"
  provision_size="$(wc -c < "$PROVISION_JSON" | tr -d '[:space:]')"
  if [[ "$provision_size" -gt "$PROVISION_JSON_MAX_BYTES" ]]; then
    echo "Relay provision JSON is too large." >&2
    exit 1
  fi
fi

if [[ -s "$PROVISION_JSON" ]]; then
  provision_package_info="$(python3 - "$PROVISION_JSON" "$pkg_arch" <<'PY'
import json, sys
j=json.load(open(sys.argv[1]))
arch=sys.argv[2]
urls=j.get('macPackageUrls') or {}
sha_urls=j.get('macPackageSha256Urls') or {}
print(urls.get(arch) or j.get('macPackageUrl') or '')
print(sha_urls.get(arch) or j.get('macPackageSha256Url') or j.get('packageSha256Url') or '')
PY
)"
  package_from_provision="$(printf '%s\n' "$provision_package_info" | sed -n '1p')"
  package_sha_from_provision="$(printf '%s\n' "$provision_package_info" | sed -n '2p')"
  if [[ -n "$package_from_provision" ]]; then PACKAGE_URL="$package_from_provision"; fi
  if [[ -n "$package_sha_from_provision" ]]; then PACKAGE_SHA256_URL="$package_sha_from_provision"; fi
fi

MAX_PMSET_OUTPUT_BYTES=65536

limited_pmset_output() {
  { pmset -g custom 2>/dev/null || pmset -g 2>/dev/null || true; } | dd bs="$MAX_PMSET_OUTPUT_BYTES" count=1 2>/dev/null || true
}

check_power_warning() {
  if ! command -v pmset >/dev/null 2>&1; then
    echo "Warning: could not check macOS power settings. Please make sure this Mac never sleeps and is not shut down during remote tasks." >&2
    return 0
  fi
  local out bad
  out="$(limited_pmset_output)"
  bad=""
  if echo "$out" | awk '/^[[:space:]]*sleep[[:space:]]+[1-9][0-9]*/{bad=1} END{exit bad?0:1}'; then
    bad="${bad}
- macOS sleep timeout is not Never"
  fi
  if echo "$out" | awk '/^[[:space:]]*(standby|autopoweroff)[[:space:]]+[1-9][0-9]*/{bad=1} END{exit bad?0:1}'; then
    bad="${bad}
- macOS standby/autopoweroff is enabled"
  fi
  if [[ -n "$bad" ]]; then
    cat <<EOF
WARNING: Gardener remote access requires this Mac to stay awake, online, and powered on.$bad
Set System Settings > Battery / Lock Screen sleep options to Never, do not close the lid during remote tasks, and do not shut down the Mac.
Optional command: sudo pmset -a sleep 0 disksleep 0 standby 0 autopoweroff 0
EOF
  fi
}

check_power_warning

shell_quote() {
  python3 -c 'import shlex, sys; print(shlex.quote(sys.argv[1]), end="")' "$1"
}

xml_escape() {
  python3 -c 'import html, sys; print(html.escape(sys.argv[1], quote=True), end="")' "${1:-}"
}

if [[ -z "$PACKAGE_URL" ]]; then
  echo "Package URL is not configured. Re-run with --relay-base-url http://YOUR_SERVER or set GARDENER_RELAY_BASE_URL." >&2
  exit 1
fi

echo "Installing Gardener to the configured install directory"
mkdir -p "$INSTALL_DIR"

echo "Downloading Gardener package..."
curl -fL --connect-timeout 20 --max-time 300 "$PACKAGE_URL" -o "$TMP/gardener.tar.gz"
verify_sha256_file "$TMP/gardener.tar.gz" "$PACKAGE_SHA256_URL" "$TMP/gardener.tar.gz.sha256"
mkdir -p "$TMP/extract"
safe_extract_tar "$TMP/gardener.tar.gz" "$TMP/extract"
SRC="$(find "$TMP/extract" -maxdepth 1 -type d -name 'Gardener-macOS-*' | head -n 1)"
if [[ -z "$SRC" ]]; then SRC="$TMP/extract"; fi

# Stop existing services before replacing files.
uid="$(id -u)"
if launchctl print "gui/$uid/com.gardener.local" >/dev/null 2>&1; then
  launchctl bootout "gui/$uid" "$HOME/Library/LaunchAgents/com.gardener.local.plist" 2>/dev/null || true
fi
if launchctl print "gui/$uid/com.gardener.relay" >/dev/null 2>&1; then
  launchctl bootout "gui/$uid" "$HOME/Library/LaunchAgents/com.gardener.relay.plist" 2>/dev/null || true
fi

backup="$INSTALL_DIR/backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$backup"
for name in gardener frpc web start-gardener.sh update-gardener.sh README-macOS.txt; do
  if [[ -e "$INSTALL_DIR/$name" ]]; then mv "$INSTALL_DIR/$name" "$backup/$name"; fi
done
for name in gardener frpc web start-gardener.sh update-gardener.sh README-macOS.txt; do
  if [[ -e "$SRC/$name" ]]; then cp -R "$SRC/$name" "$INSTALL_DIR/$name"; fi
done
chmod +x "$INSTALL_DIR/gardener" "$INSTALL_DIR/frpc" "$INSTALL_DIR/start-gardener.sh" "$INSTALL_DIR/update-gardener.sh" 2>/dev/null || true

if [[ -s "$PROVISION_JSON" ]]; then
  echo "Writing relay configuration..."
  python3 - "$PROVISION_JSON" "$INSTALL_DIR" <<'PY'
import json, pathlib, sys, datetime
provision_path, install_dir = sys.argv[1:3]
j=json.load(open(provision_path))
root=pathlib.Path(install_dir)
frpc_path=root/'frpc.toml'
frpc_path.write_text(j['frpcToml'], encoding='utf-8')
frpc_path.chmod(0o600)
relay={
  'schemaVersion': 1,
  'user': j.get('user',''),
  'publicUrl': j.get('publicUrl',''),
  'webUsername': j.get('webUsername',''),
  'webPassword': j.get('webPassword',''),
  'installedAt': datetime.datetime.now(datetime.timezone.utc).isoformat(),
}
relay_path=root/'gardener.relay.json'
relay_path.write_text(json.dumps(relay, ensure_ascii=False, indent=2)+'\n', encoding='utf-8')
relay_path.chmod(0o600)
PY
fi

cat > "$INSTALL_DIR/gardener.config.sh" <<EOF
# Generated by install-gardener.sh. You usually do not need to edit this file.
export PATH=$(shell_quote "$INSTALL_PATH")
export HOME=$(shell_quote "$HOME")
export USER=$(shell_quote "$(whoami)")
export AUTO_GARDENER_ADDR=$(shell_quote "127.0.0.1:8080")
export AUTO_GARDENER_STATIC=$(shell_quote "$INSTALL_DIR/web/static")
export AUTO_GARDENER_DATA=$(shell_quote "$HOME/Desktop/forest_data")
EOF
if [[ -n "$CODEX_CMD" ]]; then printf 'export AUTO_GARDENER_CODEX_CMD=%s\n' "$(shell_quote "$CODEX_CMD")" >> "$INSTALL_DIR/gardener.config.sh"; fi
if [[ -n "$CLAUDE_CMD" ]]; then printf 'export AUTO_GARDENER_CLAUDE_CMD=%s\n' "$(shell_quote "$CLAUDE_CMD")" >> "$INSTALL_DIR/gardener.config.sh"; fi

mkdir -p "$HOME/Library/LaunchAgents"
LOG_DIR="$INSTALL_DIR/logs"
mkdir -p "$LOG_DIR"
chmod 700 "$LOG_DIR" 2>/dev/null || true

plist_install_dir="$(xml_escape "$INSTALL_DIR")"
plist_install_path="$(xml_escape "$INSTALL_PATH")"
plist_home="$(xml_escape "$HOME")"
plist_user="$(xml_escape "$(whoami)")"
plist_codex_cmd="$(xml_escape "$CODEX_CMD")"
plist_claude_cmd="$(xml_escape "$CLAUDE_CMD")"
plist_log_dir="$(xml_escape "$LOG_DIR")"
cat > "$HOME/Library/LaunchAgents/com.gardener.local.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.gardener.local</string>
  <key>ProgramArguments</key><array><string>$plist_install_dir/gardener</string></array>
  <key>WorkingDirectory</key><string>$plist_install_dir</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key><string>$plist_install_path</string>
    <key>HOME</key><string>$plist_home</string>
    <key>USER</key><string>$plist_user</string>
    <key>AUTO_GARDENER_ADDR</key><string>127.0.0.1:8080</string>
    <key>AUTO_GARDENER_STATIC</key><string>$plist_install_dir/web/static</string>
    <key>AUTO_GARDENER_DATA</key><string>$plist_home/Desktop/forest_data</string>
EOF
if [[ -n "$CODEX_CMD" ]]; then
cat >> "$HOME/Library/LaunchAgents/com.gardener.local.plist" <<EOF
    <key>AUTO_GARDENER_CODEX_CMD</key><string>$plist_codex_cmd</string>
EOF
fi
if [[ -n "$CLAUDE_CMD" ]]; then
cat >> "$HOME/Library/LaunchAgents/com.gardener.local.plist" <<EOF
    <key>AUTO_GARDENER_CLAUDE_CMD</key><string>$plist_claude_cmd</string>
EOF
fi
cat >> "$HOME/Library/LaunchAgents/com.gardener.local.plist" <<EOF
  </dict>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>$plist_log_dir/gardener.local.out.log</string>
  <key>StandardErrorPath</key><string>$plist_log_dir/gardener.local.err.log</string>
</dict>
</plist>
EOF

if [[ -f "$INSTALL_DIR/frpc.toml" && -x "$INSTALL_DIR/frpc" ]]; then
cat > "$HOME/Library/LaunchAgents/com.gardener.relay.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.gardener.relay</string>
  <key>ProgramArguments</key><array><string>$plist_install_dir/frpc</string><string>-c</string><string>$plist_install_dir/frpc.toml</string></array>
  <key>WorkingDirectory</key><string>$plist_install_dir</string>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>$plist_log_dir/gardener.relay.out.log</string>
  <key>StandardErrorPath</key><string>$plist_log_dir/gardener.relay.err.log</string>
</dict>
</plist>
EOF
fi

if [[ "$START_AFTER_INSTALL" == "1" ]]; then
  launchctl bootstrap "gui/$uid" "$HOME/Library/LaunchAgents/com.gardener.local.plist" 2>/dev/null || true
  launchctl kickstart -k "gui/$uid/com.gardener.local" 2>/dev/null || true
  if [[ -f "$HOME/Library/LaunchAgents/com.gardener.relay.plist" ]]; then
    launchctl bootstrap "gui/$uid" "$HOME/Library/LaunchAgents/com.gardener.relay.plist" 2>/dev/null || true
    launchctl kickstart -k "gui/$uid/com.gardener.relay" 2>/dev/null || true
  fi
fi

local_url="http://127.0.0.1:8080"
public_url=""
web_user=""
web_pass=""
if [[ -f "$INSTALL_DIR/gardener.relay.json" ]]; then
  candidate_url="$(python3 - "$INSTALL_DIR/gardener.relay.json" <<'PY'
import json,sys
j=json.load(open(sys.argv[1])); print(j.get('publicUrl',''))
PY
)"
  if [[ "$candidate_url" == http://* || "$candidate_url" == https://* ]]; then
    public_url="$candidate_url"
  elif [[ -n "$candidate_url" ]]; then
    echo "Warning: relay publicUrl is not http(s); refusing to open it automatically." >&2
  fi
  web_user="$(python3 - "$INSTALL_DIR/gardener.relay.json" <<'PY'
import json,sys
j=json.load(open(sys.argv[1])); print(j.get('webUsername',''))
PY
)"
  web_pass="$(python3 - "$INSTALL_DIR/gardener.relay.json" <<'PY'
import json,sys
j=json.load(open(sys.argv[1])); print(j.get('webPassword',''))
PY
)"
fi

echo "Gardener installed."
echo "Local URL:  $local_url"
if [[ -n "$public_url" ]]; then
  echo "Remote URL: $public_url"
  echo "Login user: $web_user"
  echo "Login password: saved in gardener.relay.json; keep that file private."
fi
if [[ "$START_AFTER_INSTALL" == "1" ]]; then
  sleep 2
  open -- "${public_url:-$local_url}" >/dev/null 2>&1 || true
fi
