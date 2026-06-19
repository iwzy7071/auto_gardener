Gardener for Linux (Ubuntu)
===========================

Install from relay server:

  curl -fsSL http://YOUR_RELAY_SERVER/downloads/install-gardener-linux.sh -o install-gardener-linux.sh \
    && bash install-gardener-linux.sh --relay-base-url http://YOUR_RELAY_SERVER --setup-key YOUR_SETUP_KEY

Default install directory:

  ~/.local/share/Gardener

Default data directory:

  ~/Desktop/forest_data if ~/Desktop exists, otherwise ~/forest_data

Services are installed as systemd user services with Restart=always. When available, systemd-inhibit is used so the machine does not idle-sleep while Gardener or the relay tunnel is running:

  systemctl --user status gardener.local.service
  systemctl --user status gardener.relay.service
  journalctl --user -u gardener.local.service -f

Useful commands:

  ~/.local/share/Gardener/start-gardener.sh
  ~/.local/share/Gardener/update-gardener.sh --relay-base-url http://YOUR_RELAY_SERVER --setup-key YOUR_SETUP_KEY

The installer tries to enable lingering so user services can start after reboot without an interactive login. If automatic lingering fails, run:

  loginctl enable-linger "$USER"

If systemd user services are unavailable, start-gardener.sh falls back to nohup processes for both Gardener and frpc.
