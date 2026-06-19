#!/usr/bin/env bash
set -euo pipefail
INSTALL_DIR="${GARDENER_INSTALL_DIR:-$HOME/.local/share/Gardener}"
SCRIPT="$INSTALL_DIR/install-gardener.sh"
if [[ ! -x "$SCRIPT" ]]; then
  echo "Installer not found at $SCRIPT" >&2
  exit 1
fi
exec "$SCRIPT" "$@"
