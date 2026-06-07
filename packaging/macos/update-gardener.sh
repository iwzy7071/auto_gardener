#!/usr/bin/env bash
set -euo pipefail
RELAY_BASE_URL="${GARDENER_RELAY_BASE_URL:-}"
INSTALL_DIR="$HOME/Applications/Gardener"
START_AFTER_UPDATE=1
PACKAGE_URL=""
PACKAGE_MAX_BYTES=$((1024 * 1024 * 1024))
while [[ $# -gt 0 ]]; do
  case "$1" in
    --package-url) PACKAGE_URL="${2:-}"; shift 2 ;;
    --install-dir) INSTALL_DIR="${2:-}"; shift 2 ;;
    --relay-base-url) RELAY_BASE_URL="${2:-}"; shift 2 ;;
    --no-start) START_AFTER_UPDATE=0; shift ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done
arch="$(uname -m)"; [[ "$arch" == "x86_64" ]] && arch="amd64"
if [[ -z "$PACKAGE_URL" ]]; then
  if [[ -z "$RELAY_BASE_URL" ]]; then
    echo "Package URL is not configured. Re-run with --package-url URL, --relay-base-url URL, or set GARDENER_RELAY_BASE_URL." >&2
    exit 1
  fi
  PACKAGE_URL="${RELAY_BASE_URL%/}/downloads/Gardener-macOS-$arch.tar.gz"
fi
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
curl -fL --connect-timeout 20 --max-time 300 --max-filesize "$PACKAGE_MAX_BYTES" "$PACKAGE_URL" -o "$TMP/gardener.tar.gz"
mkdir -p "$TMP/extract"; tar -xzf "$TMP/gardener.tar.gz" -C "$TMP/extract"
SRC="$(find "$TMP/extract" -maxdepth 1 -type d -name 'Gardener-macOS-*' | head -n 1)"; [[ -z "$SRC" ]] && SRC="$TMP/extract"
uid="$(id -u)"
launchctl bootout "gui/$uid" "$HOME/Library/LaunchAgents/com.gardener.local.plist" 2>/dev/null || true
launchctl bootout "gui/$uid" "$HOME/Library/LaunchAgents/com.gardener.relay.plist" 2>/dev/null || true
backup="$INSTALL_DIR/backup-$(date +%Y%m%d-%H%M%S)"; mkdir -p "$backup"
for name in gardener frpc web start-gardener.sh update-gardener.sh README-macOS.txt; do [[ -e "$INSTALL_DIR/$name" ]] && mv "$INSTALL_DIR/$name" "$backup/$name"; done
for name in gardener frpc web start-gardener.sh update-gardener.sh README-macOS.txt; do [[ -e "$SRC/$name" ]] && cp -R "$SRC/$name" "$INSTALL_DIR/$name"; done
chmod +x "$INSTALL_DIR/gardener" "$INSTALL_DIR/frpc" "$INSTALL_DIR/start-gardener.sh" "$INSTALL_DIR/update-gardener.sh" 2>/dev/null || true
if [[ "$START_AFTER_UPDATE" == "1" ]]; then
  launchctl bootstrap "gui/$uid" "$HOME/Library/LaunchAgents/com.gardener.local.plist" 2>/dev/null || true
  launchctl kickstart -k "gui/$uid/com.gardener.local" 2>/dev/null || true
  [[ -f "$HOME/Library/LaunchAgents/com.gardener.relay.plist" ]] && launchctl bootstrap "gui/$uid" "$HOME/Library/LaunchAgents/com.gardener.relay.plist" 2>/dev/null || true
  [[ -f "$HOME/Library/LaunchAgents/com.gardener.relay.plist" ]] && launchctl kickstart -k "gui/$uid/com.gardener.relay" 2>/dev/null || true
fi
echo "Gardener updated. Backup: $backup"
