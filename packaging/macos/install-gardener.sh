#!/usr/bin/env bash
set -euo pipefail

RELAY_BASE_URL="${GARDENER_RELAY_BASE_URL:-}"
INSTALL_DIR="$HOME/Applications/Gardener"
SETUP_KEY=""
PROVISION_URL=""
START_AFTER_INSTALL=1

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
    --start|--start-after-install) START_AFTER_INSTALL=1; shift ;;
    --no-start) START_AFTER_INSTALL=0; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

RELAY_BASE_URL="${RELAY_BASE_URL%/}"
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
INSTALL_PATH="$PATH:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/local/sbin:$HOME/.local/bin"
CODEX_CMD="$(PATH="$INSTALL_PATH" command -v codex || true)"
CLAUDE_CMD="$(PATH="$INSTALL_PATH" command -v claude || true)"

TMP="$(mktemp -d)"
cleanup(){ rm -rf "$TMP"; }
trap cleanup EXIT

PROVISION_JSON="$TMP/provision.json"
if [[ -n "$PROVISION_URL" ]]; then
  echo "Loading Gardener relay provision..."
  curl -fsSL --connect-timeout 20 --max-time 120 "$PROVISION_URL" -o "$PROVISION_JSON"
fi

if [[ -s "$PROVISION_JSON" ]]; then
  package_from_provision="$(python3 - "$PROVISION_JSON" "$pkg_arch" <<'PY'
import json, sys
j=json.load(open(sys.argv[1]))
arch=sys.argv[2]
urls=j.get('macPackageUrls') or {}
print(urls.get(arch) or j.get('macPackageUrl') or '')
PY
)"
  if [[ -n "$package_from_provision" ]]; then PACKAGE_URL="$package_from_provision"; fi
fi

check_power_warning() {
  if ! command -v pmset >/dev/null 2>&1; then
    echo "Warning: could not check macOS power settings. Please make sure this Mac never sleeps and is not shut down during remote tasks." >&2
    return 0
  fi
  local out bad
  out="$(pmset -g custom 2>/dev/null || pmset -g 2>/dev/null || true)"
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

if [[ -z "$PACKAGE_URL" ]]; then
  echo "Package URL is not configured. Re-run with --relay-base-url http://YOUR_SERVER or set GARDENER_RELAY_BASE_URL." >&2
  exit 1
fi

echo "Installing Gardener to $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"

echo "Downloading Gardener package..."
curl -fL --connect-timeout 20 --max-time 300 "$PACKAGE_URL" -o "$TMP/gardener.tar.gz"
mkdir -p "$TMP/extract"
tar -xzf "$TMP/gardener.tar.gz" -C "$TMP/extract"
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
  python3 - "$PROVISION_JSON" "$INSTALL_DIR" "$PROVISION_URL" <<'PY'
import json, pathlib, sys, datetime
provision_path, install_dir, provision_url = sys.argv[1:4]
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
  'provisionUrl': provision_url,
  'installedAt': datetime.datetime.now(datetime.timezone.utc).isoformat(),
}
relay_path=root/'gardener.relay.json'
relay_path.write_text(json.dumps(relay, ensure_ascii=False, indent=2)+'\n', encoding='utf-8')
relay_path.chmod(0o600)
PY
fi

cat > "$INSTALL_DIR/gardener.config.sh" <<EOF
# Generated by install-gardener.sh. You usually do not need to edit this file.
export PATH="$INSTALL_PATH"
export HOME="$HOME"
export USER="$(whoami)"
export AUTO_GARDENER_ADDR="127.0.0.1:8080"
export AUTO_GARDENER_STATIC="$INSTALL_DIR/web/static"
export AUTO_GARDENER_DATA="$HOME/Desktop/forest_data"
EOF
if [[ -n "$CODEX_CMD" ]]; then printf 'export AUTO_GARDENER_CODEX_CMD="%s"\n' "$CODEX_CMD" >> "$INSTALL_DIR/gardener.config.sh"; fi
if [[ -n "$CLAUDE_CMD" ]]; then printf 'export AUTO_GARDENER_CLAUDE_CMD="%s"\n' "$CLAUDE_CMD" >> "$INSTALL_DIR/gardener.config.sh"; fi

mkdir -p "$HOME/Library/LaunchAgents"
cat > "$HOME/Library/LaunchAgents/com.gardener.local.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.gardener.local</string>
  <key>ProgramArguments</key><array><string>$INSTALL_DIR/gardener</string></array>
  <key>WorkingDirectory</key><string>$INSTALL_DIR</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key><string>$INSTALL_PATH</string>
    <key>HOME</key><string>$HOME</string>
    <key>USER</key><string>$(whoami)</string>
    <key>AUTO_GARDENER_ADDR</key><string>127.0.0.1:8080</string>
    <key>AUTO_GARDENER_STATIC</key><string>$INSTALL_DIR/web/static</string>
    <key>AUTO_GARDENER_DATA</key><string>$HOME/Desktop/forest_data</string>
EOF
if [[ -n "$CODEX_CMD" ]]; then
cat >> "$HOME/Library/LaunchAgents/com.gardener.local.plist" <<EOF
    <key>AUTO_GARDENER_CODEX_CMD</key><string>$CODEX_CMD</string>
EOF
fi
if [[ -n "$CLAUDE_CMD" ]]; then
cat >> "$HOME/Library/LaunchAgents/com.gardener.local.plist" <<EOF
    <key>AUTO_GARDENER_CLAUDE_CMD</key><string>$CLAUDE_CMD</string>
EOF
fi
cat >> "$HOME/Library/LaunchAgents/com.gardener.local.plist" <<EOF
  </dict>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>/tmp/gardener.local.out.log</string>
  <key>StandardErrorPath</key><string>/tmp/gardener.local.err.log</string>
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
  <key>ProgramArguments</key><array><string>$INSTALL_DIR/frpc</string><string>-c</string><string>$INSTALL_DIR/frpc.toml</string></array>
  <key>WorkingDirectory</key><string>$INSTALL_DIR</string>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>/tmp/gardener.relay.out.log</string>
  <key>StandardErrorPath</key><string>/tmp/gardener.relay.err.log</string>
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
  open "${public_url:-$local_url}" >/dev/null 2>&1 || true
fi
