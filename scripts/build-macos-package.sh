#!/usr/bin/env bash
set -euo pipefail
VERSION="${VERSION:-dev}"
OUT_DIR="${OUT_DIR:-dist}"
FRP_VERSION="${FRP_VERSION:-0.52.3}"
if (( ${#FRP_VERSION} > 32 )) || [[ ! "$FRP_VERSION" =~ ^[0-9]+(\.[0-9]+){1,3}$ ]]; then
  echo "FRP_VERSION must be a dotted numeric version such as 0.52.3" >&2
  exit 1
fi
ARCHES="${ARCHES:-arm64 amd64}"
mkdir -p "$OUT_DIR"

fetch_frpc() {
  local arch="$1"
  local cache="packaging/macos/frpc-darwin-${arch}"
  if [[ -x "$cache" ]]; then
    echo "$cache"
    return 0
  fi
  local url_arch="$arch"
  local tmp
  tmp="$(mktemp -d)"
  local url="https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_darwin_${url_arch}.tar.gz"
  if [[ "${DOWNLOAD_FRPC:-0}" == "1" ]]; then
    echo "Downloading frpc for darwin/$arch from $url" >&2
    if ! curl -L --fail --connect-timeout 20 --max-time 240 -o "$tmp/frp.tgz" "$url"; then
      url="https://gh-proxy.com/https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_darwin_${url_arch}.tar.gz"
      curl -L --fail --connect-timeout 20 --max-time 240 -o "$tmp/frp.tgz" "$url"
    fi
    tar -xzf "$tmp/frp.tgz" -C "$tmp"
    local found
    found="$(find "$tmp" -name frpc -type f | head -n 1)"
    if [[ -z "$found" ]]; then echo "frpc not found for $arch" >&2; exit 1; fi
    cp "$found" "$cache"
    chmod +x "$cache"
    rm -rf "$tmp"
    echo "$cache"
    return 0
  fi
  rm -rf "$tmp"
  echo ""
}

for arch in $ARCHES; do
  case "$arch" in
    arm64) goarch="arm64" ;;
    amd64|x86_64) arch="amd64"; goarch="amd64" ;;
    *) echo "Unsupported arch: $arch" >&2; exit 1 ;;
  esac
  pkg_dir="$OUT_DIR/Gardener-macOS-$arch"
  tar_path="$OUT_DIR/Gardener-macOS-$arch.tar.gz"
  rm -rf "$pkg_dir" "$tar_path"
  mkdir -p "$pkg_dir/web"
  GOOS=darwin GOARCH="$goarch" go build -ldflags "-X auto_gardener/internal/app.Version=$VERSION" -o "$pkg_dir/gardener" ./cmd/server
  cp -R web/static "$pkg_dir/web/static"
  cp packaging/macos/install-gardener.sh "$pkg_dir/install-gardener.sh"
  cp packaging/macos/start-gardener.sh "$pkg_dir/start-gardener.sh"
  cp packaging/macos/update-gardener.sh "$pkg_dir/update-gardener.sh"
  cp packaging/macos/README-macOS.txt "$pkg_dir/README-macOS.txt"
  chmod +x "$pkg_dir/gardener" "$pkg_dir/install-gardener.sh" "$pkg_dir/start-gardener.sh" "$pkg_dir/update-gardener.sh"
  frpc_path="$(fetch_frpc "$arch")"
  if [[ -n "$frpc_path" && -f "$frpc_path" ]]; then
    cp "$frpc_path" "$pkg_dir/frpc"
    chmod +x "$pkg_dir/frpc"
  else
    echo "Warning: frpc not included for macOS $arch. Set DOWNLOAD_FRPC=1 or place packaging/macos/frpc-darwin-$arch." >&2
  fi
  printf '%s\n' "$VERSION" > "$pkg_dir/VERSION.txt"
  ( cd "$OUT_DIR" && tar -czf "Gardener-macOS-$arch.tar.gz" "Gardener-macOS-$arch" )
  echo "Built $tar_path"
done
