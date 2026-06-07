Gardener for macOS

Recommended one-command relay install:

  curl -fsSL http://YOUR_RELAY_SERVER/downloads/install-gardener-macos.sh -o install-gardener-macos.sh && bash install-gardener-macos.sh --relay-base-url http://YOUR_RELAY_SERVER --setup-key YOUR_SETUP_KEY

The installer will:
1. Download the correct macOS package for Apple Silicon or Intel Mac.
2. Install Gardener into ~/Applications/Gardener.
3. Write gardener.config.sh, frpc.toml and gardener.relay.json from your SetupKey.
4. Register LaunchAgents com.gardener.local and com.gardener.relay.
5. Start local Gardener at http://127.0.0.1:8080 and the remote relay tunnel.

Start later:

  ~/Applications/Gardener/start-gardener.sh

Update later:

  ~/Applications/Gardener/update-gardener.sh

Data is stored at ~/Desktop/forest_data by default.
