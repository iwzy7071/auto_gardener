Gardener for Windows

Recommended one-click relay install

Ask the administrator for your SetupKey (starts with sk_). Then run this in PowerShell:

  powershell -ExecutionPolicy Bypass -Command "iwr http://YOUR_RELAY_SERVER/downloads/install-gardener.ps1 -OutFile install-gardener.ps1; .\install-gardener.ps1 -RelayBaseUrl http://YOUR_RELAY_SERVER -SetupKey YOUR_SETUP_KEY -DesktopShortcut -StartMenuShortcut -StartAfterInstall"

After installation, double click Gardener. It will:
1. Start local Gardener at http://127.0.0.1:8080
2. Start the relay tunnel automatically if frpc.exe and frpc.toml exist
3. Open your assigned remote URL
4. Keep the PowerShell window alive; if gardener.exe exits unexpectedly, start-gardener.ps1 will restart it automatically.

The remote URL requires the username saved in gardener.relay.json and the password provided by the relay administrator.

Local-only start

Double click start-gardener.bat to run Gardener locally.
Browser opens http://127.0.0.1:8080 by default.
Local data is stored at Desktop\forest_data unless configured otherwise.
To customize settings, copy gardener.config.example.ps1 to gardener.config.ps1 and edit it.

Windows security prompt note

The installer/update script removes the Internet download mark from Gardener files and forces Gardener to listen on 127.0.0.1:8080 only. This prevents the common Windows Defender Firewall prompt such as "some features of this app have been blocked" while still allowing browser access and the relay tunnel. If you run the installer as Administrator, it also adds a firewall rule that blocks external inbound access to gardener.exe.

Do not double click gardener.exe directly; use the Gardener shortcut or start-gardener.bat so the local-only configuration is loaded.

Upgrade

Run:
  powershell -ExecutionPolicy Bypass -File update-gardener.ps1 -PackageUrl "http://YOUR_RELAY_SERVER/downloads/Gardener-Windows.zip" -Restart

Upgrades keep gardener.config.ps1, gardener.relay.json, frpc.toml and Desktop\forest_data.
Do not delete Desktop\forest_data during upgrades.
