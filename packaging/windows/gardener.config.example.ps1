# Copy this file to gardener.config.ps1 and edit values as needed.
# Local service address. Use 127.0.0.1 for local-only + tunnel/relay access.
$env:AUTO_GARDENER_ADDR = "127.0.0.1:8080"

# Optional: pin data/static directories.
# $env:AUTO_GARDENER_DATA = "D:\forest_data"
# $env:AUTO_GARDENER_STATIC = "C:\Gardener\web\static"

# Optional: CLI commands.
# $env:AUTO_GARDENER_CODEX_CMD = "$env:APPDATA\npm\codex.cmd"
# $env:AUTO_GARDENER_CLAUDE_CMD = "$env:APPDATA\npm\claude.cmd"

# Optional: DingTalk incoming signature secret.
# $env:AUTO_GARDENER_DINGTALK_INCOMING_SECRET = "your_dingtalk_app_secret"
