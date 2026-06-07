# Security Policy

Gardener is local-first software that can run CLI tools, open network tunnels, and modify files in user-selected workspaces. Treat every security report seriously.

## Supported versions

Until the project adopts formal releases, security fixes target the `main` branch.

## Reporting a vulnerability

Please do **not** publish exploit details in a public issue.

Preferred options:

1. Use GitHub private vulnerability reporting / security advisories if it is enabled for this repository.
2. If private reporting is unavailable, open a public issue with only a high-level summary and request a private contact path. Do not include secrets, working exploits, tokens, passwords, or server addresses.

## What to include privately

- Affected commit, version, or installer package.
- Operating system and installation method.
- Minimal reproduction steps.
- Impact assessment.
- Suggested mitigation, if known.

## Secret handling

Never commit or paste:

- SSH private keys, passwords, Basic Auth credentials, setup keys, or API tokens.
- Real relay server addresses or user-specific frp configuration.
- `frpc.toml`, `htpasswd`, provision JSON, runtime `settings.json`, `usage.jsonl`, task workspaces, or packaged binaries.

See [`SECURITY_PUBLIC_RELEASE.md`](SECURITY_PUBLIC_RELEASE.md) for the public-release checklist.
