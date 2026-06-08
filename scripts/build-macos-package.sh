#!/usr/bin/env bash
set -euo pipefail
VERSION="${VERSION:-dev}"
if (( ${#VERSION} > 64 )) || [[ ! "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._+-]*$ ]]; then
  echo "VERSION must be 1-64 characters and contain only letters, digits, dot, underscore, plus or dash" >&2
  exit 1
fi
OUT_DIR="${OUT_DIR:-dist}"
case "$OUT_DIR" in
  ""|"/"|"."|".."|../*|*/../*)
    echo "Refusing unsafe OUT_DIR: $OUT_DIR" >&2
    exit 1
    ;;
esac
FRP_VERSION="${FRP_VERSION:-0.52.3}"
ARCHES="${ARCHES:-arm64 amd64}"

if (( ${#ARCHES} > 64 )); then
  echo "ARCHES is too long" >&2
  exit 1
fi

normalize_arches() {
  local normalized=()
  local arch canonical existing
  for arch in $ARCHES; do
    case "$arch" in
      arm64) canonical="arm64" ;;
      amd64|x86_64) canonical="amd64" ;;
      *) echo "Unsupported arch: $arch" >&2; exit 1 ;;
    esac
    for existing in "${normalized[@]}"; do
      if [[ "$existing" == "$canonical" ]]; then
        canonical=""
        break
      fi
    done
    if [[ -n "$canonical" ]]; then
      normalized+=("$canonical")
      if (( ${#normalized[@]} > 2 )); then
        echo "Too many macOS architectures requested" >&2
        exit 1
      fi
    fi
  done
  if (( ${#normalized[@]} == 0 )); then
    echo "No macOS architectures requested" >&2
    exit 1
  fi
  printf '%s\n' "${normalized[@]}"
}

mapfile -t BUILD_ARCHES < <(normalize_arches)
mkdir -p "$OUT_DIR"

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
  trap 'rm -rf "$tmp"' RETURN
  local url="https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_darwin_${url_arch}.tar.gz"
  if [[ "${DOWNLOAD_FRPC:-0}" == "1" ]]; then
    echo "Downloading frpc for darwin/$arch..." >&2
    if ! curl -L --fail --connect-timeout 20 --max-time 240 -o "$tmp/frp.tgz" "$url"; then
      url="https://gh-proxy.com/https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_darwin_${url_arch}.tar.gz"
      curl -L --fail --connect-timeout 20 --max-time 240 -o "$tmp/frp.tgz" "$url"
    fi
    safe_extract_tar "$tmp/frp.tgz" "$tmp"
    local found
    found="$tmp/frp_${FRP_VERSION}_darwin_${url_arch}/frpc"
    if [[ ! -f "$found" ]]; then echo "expected frpc not found for $arch" >&2; exit 1; fi
    cp "$found" "$cache"
    chmod +x "$cache"
    rm -rf "$tmp"
    echo "$cache"
    return 0
  fi
  rm -rf "$tmp"
  echo ""
}

for arch in "${BUILD_ARCHES[@]}"; do
  case "$arch" in
    arm64) goarch="arm64" ;;
    amd64) goarch="amd64" ;;
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
  ( cd "$OUT_DIR" && sha256sum "Gardener-macOS-$arch.tar.gz" > "Gardener-macOS-$arch.tar.gz.sha256" )
  echo "Built $tar_path"
done
