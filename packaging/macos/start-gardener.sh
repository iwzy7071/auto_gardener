#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
uid="$(id -u)"

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
launchctl bootstrap "gui/$uid" "$HOME/Library/LaunchAgents/com.gardener.local.plist" 2>/dev/null || true
launchctl kickstart -k "gui/$uid/com.gardener.local" 2>/dev/null || true
if [[ -f "$ROOT/frpc.toml" && -f "$HOME/Library/LaunchAgents/com.gardener.relay.plist" ]]; then
  launchctl bootstrap "gui/$uid" "$HOME/Library/LaunchAgents/com.gardener.relay.plist" 2>/dev/null || true
  launchctl kickstart -k "gui/$uid/com.gardener.relay" 2>/dev/null || true
fi
url="http://127.0.0.1:8080"
if [[ -f "$ROOT/gardener.relay.json" ]]; then
  u="$(python3 - "$ROOT/gardener.relay.json" <<'PY'
import json,sys
print(json.load(open(sys.argv[1])).get('publicUrl',''))
PY
)"
  if [[ "$u" == http://* || "$u" == https://* ]]; then
    url="$u"
  elif [[ -n "$u" ]]; then
    echo "Warning: relay publicUrl is not http(s); refusing to open it automatically." >&2
  fi
fi
open -- "$url" >/dev/null 2>&1 || true
echo "Gardener started: $url"
