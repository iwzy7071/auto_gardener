#!/usr/bin/env bash
set -euo pipefail
INSTALL_DIR="${GARDENER_INSTALL_DIR:-$HOME/.local/share/Gardener}"
if [[ -f "$INSTALL_DIR/gardener.config.sh" ]]; then
  # shellcheck disable=SC1091
  source "$INSTALL_DIR/gardener.config.sh"
fi
if command -v systemctl >/dev/null 2>&1; then
  systemctl --user daemon-reload >/dev/null 2>&1 || true
  systemctl --user enable --now gardener.local.service >/dev/null 2>&1 || true
  if [[ -f "$INSTALL_DIR/frpc.toml" && -x "$INSTALL_DIR/frpc" ]]; then
    systemctl --user enable --now gardener.relay.service >/dev/null 2>&1 || true
  fi
fi
mkdir -p "$INSTALL_DIR/logs"
if ! pgrep -u "$(id -u)" -f "$INSTALL_DIR/gardener" >/dev/null 2>&1; then
  if command -v systemd-inhibit >/dev/null 2>&1; then
    ( cd "$INSTALL_DIR" && nohup systemd-inhibit --what=sleep --mode=block --why=GardenerRemoteAccess "$INSTALL_DIR/gardener" >> "$INSTALL_DIR/logs/gardener.local.out.log" 2>> "$INSTALL_DIR/logs/gardener.local.err.log" & )
  else
    ( cd "$INSTALL_DIR" && nohup "$INSTALL_DIR/gardener" >> "$INSTALL_DIR/logs/gardener.local.out.log" 2>> "$INSTALL_DIR/logs/gardener.local.err.log" & )
  fi
fi
if [[ -f "$INSTALL_DIR/frpc.toml" && -x "$INSTALL_DIR/frpc" ]] && ! pgrep -u "$(id -u)" -f "$INSTALL_DIR/frpc.*$INSTALL_DIR/frpc.toml" >/dev/null 2>&1; then
  if command -v systemd-inhibit >/dev/null 2>&1; then
    ( cd "$INSTALL_DIR" && nohup systemd-inhibit --what=sleep --mode=block --why=GardenerRelayTunnel "$INSTALL_DIR/frpc" -c "$INSTALL_DIR/frpc.toml" >> "$INSTALL_DIR/logs/gardener.relay.out.log" 2>> "$INSTALL_DIR/logs/gardener.relay.err.log" & )
  else
    ( cd "$INSTALL_DIR" && nohup "$INSTALL_DIR/frpc" -c "$INSTALL_DIR/frpc.toml" >> "$INSTALL_DIR/logs/gardener.relay.out.log" 2>> "$INSTALL_DIR/logs/gardener.relay.err.log" & )
  fi
fi
echo "Gardener local URL: http://127.0.0.1:8080"
