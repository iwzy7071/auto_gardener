# Public release safety checklist

This repository is intended to be safe for a public GitHub repository. Keep all deployment-specific values outside git.

## Never commit

- VPS IPs or domain names if they identify a private deployment.
- SSH usernames, passwords, private keys, setup keys, frp tokens, Basic Auth passwords.
- `frpc.toml`, `htpasswd`, provision JSON files, generated relay user folders.
- Built packages from `dist/` or binaries from `bin/`.
- Runtime task data such as `forest_data`, `settings.json`, `usage.jsonl`, logs, reports and downloaded files.

## Local configuration

Copy the example and edit the local file:

```bash
cp config/gardener-relay.env.example config/gardener-relay.env.local
$EDITOR config/gardener-relay.env.local
```

Load it before running relay deployment commands:

```bash
set -a
source config/gardener-relay.env.local
set +a
```

The installer scripts also accept explicit relay URLs:

```powershell
.\install-gardener.ps1 -RelayBaseUrl http://YOUR_RELAY_SERVER -SetupKey sk_xxx
```

```bash
bash install-gardener-macos.sh --relay-base-url http://YOUR_RELAY_SERVER --setup-key sk_xxx
```

## Before pushing

Run:

```bash
git status --ignored
grep -RInE 'PASSWORD|SECRET|TOKEN|sk_|auth\.token|YOUR_REAL_SERVER|[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' \
  --exclude-dir=.git --exclude-dir=dist --exclude-dir=dist-test --exclude-dir=bin --exclude='*.local*' .
```

The grep may show harmless code references to token fields. It must not show real secrets.
