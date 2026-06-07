#!/usr/bin/env bash
set -euo pipefail
VERSION="${VERSION:-dev}"
if (( ${#VERSION} > 64 )) || [[ ! "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._+-]*$ ]]; then
  echo "VERSION must be 1-64 characters and contain only letters, digits, dot, underscore, plus or dash" >&2
  exit 1
fi
OUT_DIR="${OUT_DIR:-dist}"
PKG_DIR="$OUT_DIR/Gardener-Windows"
ZIP_PATH="$OUT_DIR/Gardener-Windows.zip"

rm -rf "$PKG_DIR" "$ZIP_PATH"
mkdir -p "$PKG_DIR/web"

GOOS=windows GOARCH=amd64 go build -ldflags "-X auto_gardener/internal/app.Version=$VERSION" -o "$PKG_DIR/gardener.exe" ./cmd/server
cp -R web/static "$PKG_DIR/web/static"
cp packaging/windows/start-gardener.bat "$PKG_DIR/start-gardener.bat"
cp packaging/windows/start-gardener.ps1 "$PKG_DIR/start-gardener.ps1"
cp packaging/windows/update-gardener.ps1 "$PKG_DIR/update-gardener.ps1"
cp packaging/windows/install-gardener.ps1 "$PKG_DIR/install-gardener.ps1"
cp packaging/windows/gardener.config.example.ps1 "$PKG_DIR/gardener.config.example.ps1"
cp packaging/windows/README-Windows.txt "$PKG_DIR/README-Windows.txt"
cp packaging/windows/frpc.example.toml "$PKG_DIR/frpc.example.toml"

FRP_VERSION="${FRP_VERSION:-0.52.3}"
if [[ -n "${FRPC_EXE:-}" && -f "$FRPC_EXE" ]]; then
  cp "$FRPC_EXE" "$PKG_DIR/frpc.exe"
elif [[ -f packaging/windows/frpc.exe ]]; then
  cp packaging/windows/frpc.exe "$PKG_DIR/frpc.exe"
elif [[ "${DOWNLOAD_FRPC:-0}" == "1" ]]; then
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  url="https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_windows_amd64.zip"
  echo "Downloading frpc.exe from $url"
  curl -L --fail --connect-timeout 20 --max-time 240 -o "$tmp/frp.zip" "$url"
  unzip -q "$tmp/frp.zip" -d "$tmp/frp"
  found="$(find "$tmp/frp" -name frpc.exe -type f | head -n 1)"
  if [[ -z "$found" ]]; then
    echo "frpc.exe not found in downloaded archive" >&2
    exit 1
  fi
  cp "$found" "$PKG_DIR/frpc.exe"
else
  echo "Warning: frpc.exe not included. Set FRPC_EXE=/path/to/frpc.exe or DOWNLOAD_FRPC=1 for relay one-click packages." >&2
fi

printf '%s\n' "$VERSION" > "$PKG_DIR/VERSION.txt"

( cd "$OUT_DIR" && zip -qr "Gardener-Windows.zip" "Gardener-Windows" )
echo "Built $ZIP_PATH"
