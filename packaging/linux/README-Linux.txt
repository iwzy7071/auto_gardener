Gardener for Linux (Ubuntu)
===========================

Install from relay server:

  curl -fsSL http://YOUR_RELAY_SERVER/downloads/install-gardener-linux.sh -o install-gardener-linux.sh \
    && bash install-gardener-linux.sh --relay-base-url http://YOUR_RELAY_SERVER --setup-key YOUR_SETUP_KEY

Default install directory:

  ~/.local/share/Gardener

Default data directory:

  ~/Desktop/forest_data if ~/Desktop exists, otherwise ~/forest_data

Services are installed as systemd user services:

  systemctl --user status gardener.local.service
  systemctl --user status gardener.relay.service
  journalctl --user -u gardener.local.service -f

Useful commands:

  ~/.local/share/Gardener/start-gardener.sh
  ~/.local/share/Gardener/update-gardener.sh --relay-base-url http://YOUR_RELAY_SERVER --setup-key YOUR_SETUP_KEY

If systemd user services do not start after reboot, enable lingering:

  loginctl enable-linger "$USER"
