#!/usr/bin/env bash
set -euo pipefail
VERSION="${VERSION:-dev}"
if (( ${#VERSION} > 64 )) || [[ ! "$VERSION" =~ ^[A-Za-z0-9][A-Za-z0-9._+-]*$ ]]; then
  echo "VERSION must be 1-64 characters and contain only letters, digits, dot, underscore, plus or dash" >&2
  exit 1
fi
OUT_DIR="${OUT_DIR:-dist}"
case "$OUT_DIR" in ""|"/"|"."|".."|../*|*/../*) echo "Refusing unsafe OUT_DIR: $OUT_DIR" >&2; exit 1 ;; esac
FRP_VERSION="${FRP_VERSION:-0.52.3}"
ARCHES="${ARCHES:-amd64 arm64}"
mkdir -p "$OUT_DIR"

write_sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$file" > "$file.sha256"; else shasum -a 256 "$file" > "$file.sha256"; fi
}

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

fetch_frpc() {
  local arch="$1" cache="packaging/linux/frpc-linux-${arch}"
  [[ -x "$cache" ]] && { echo "$cache"; return 0; }
  if [[ "${DOWNLOAD_FRPC:-0}" != "1" ]]; then echo ""; return 0; fi
  local tmp url
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' RETURN
  url="https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_linux_${arch}.tar.gz"
  echo "Downloading frpc for linux/$arch..." >&2
  if ! curl -L --fail --connect-timeout 20 --max-time 240 -o "$tmp/frp.tgz" "$url"; then
    url="https://gh-proxy.com/https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_linux_${arch}.tar.gz"
    curl -L --fail --connect-timeout 20 --max-time 240 -o "$tmp/frp.tgz" "$url"
  fi
  safe_extract_tar "$tmp/frp.tgz" "$tmp"
  local found="$tmp/frp_${FRP_VERSION}_linux_${arch}/frpc"
  [[ -f "$found" ]] || { echo "expected frpc not found for $arch" >&2; exit 1; }
  cp "$found" "$cache"; chmod +x "$cache"; echo "$cache"
}

normalize_arch() {
  case "$1" in amd64|x86_64) echo amd64 ;; arm64|aarch64) echo arm64 ;; *) echo "Unsupported arch: $1" >&2; exit 1 ;; esac
}

seen=""
for raw_arch in $ARCHES; do
  arch="$(normalize_arch "$raw_arch")"
  case " $seen " in *" $arch "*) continue ;; esac
  seen="$seen $arch"
  pkg_dir="$OUT_DIR/Gardener-Linux-$arch"
  tar_path="$OUT_DIR/Gardener-Linux-$arch.tar.gz"
  rm -rf "$pkg_dir" "$tar_path"
  mkdir -p "$pkg_dir/web"
  GOOS=linux GOARCH="$arch" go build -ldflags "-X auto_gardener/internal/app.Version=$VERSION" -o "$pkg_dir/gardener" ./cmd/server
  cp -R web/static "$pkg_dir/web/static"
  cp packaging/linux/install-gardener.sh "$pkg_dir/install-gardener.sh"
  cp packaging/linux/start-gardener.sh "$pkg_dir/start-gardener.sh"
  cp packaging/linux/update-gardener.sh "$pkg_dir/update-gardener.sh"
  cp packaging/linux/README-Linux.txt "$pkg_dir/README-Linux.txt"
  chmod +x "$pkg_dir/gardener" "$pkg_dir/install-gardener.sh" "$pkg_dir/start-gardener.sh" "$pkg_dir/update-gardener.sh"
  frpc_path="$(fetch_frpc "$arch")"
  if [[ -n "$frpc_path" && -f "$frpc_path" ]]; then cp "$frpc_path" "$pkg_dir/frpc"; chmod +x "$pkg_dir/frpc"; else echo "Warning: frpc not included for Linux $arch. Set DOWNLOAD_FRPC=1 or place packaging/linux/frpc-linux-$arch." >&2; fi
  printf '%s\n' "$VERSION" > "$pkg_dir/VERSION.txt"
  ( cd "$OUT_DIR" && tar -czf "Gardener-Linux-$arch.tar.gz" "Gardener-Linux-$arch" )
  write_sha256_file "$tar_path"
  echo "Built $tar_path"
done
