# Contributing to Gardener

Thanks for your interest in improving Gardener. This project is local-first software that can execute CLI agents and modify files, so reliability and safety are as important as features.

## Development principles

- Keep private deployment data out of git: no server IPs, passwords, tokens, private keys, generated `frpc.toml`, `htpasswd`, runtime data, or packaged binaries.
- Keep Windows support working. Windows is a first-class target for Gardener.
- Prefer small, reviewable pull requests with tests.
- Avoid writing generated task artifacts into the repository unless the change explicitly requires fixtures.
- Use local configuration files ignored by git for machine-specific values.

## Local setup

Requirements:

- Go 1.20 or newer.
- Node.js, for checking the browser JavaScript.
- Git.

Run checks before submitting:

```bash
go test ./...
go vet ./...
node --check web/static/app.js
GOOS=windows GOARCH=amd64 go build -o /tmp/gardener.exe ./cmd/server
```

Optional package smoke tests:

```bash
OUT_DIR=dist-test ./scripts/build-windows-package.sh
OUT_DIR=dist-test ./scripts/build-macos-package.sh
```

The package scripts may omit `frpc` unless you provide local binaries or set `DOWNLOAD_FRPC=1`.

## Branch and commit style

- Use a short branch name, for example `fix/progress-timeout` or `feat/mobile-ui`.
- Use clear commit messages in the imperative mood, for example `Fix stale output panel refresh`.
- Keep unrelated changes in separate pull requests.

## Pull request checklist

Before opening a PR, confirm:

- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `node --check web/static/app.js` passes.
- [ ] Windows build still compiles.
- [ ] No local credentials, server addresses, generated packages, or runtime data are included.
- [ ] User-facing behavior is documented when it changes.

## Reporting vulnerabilities

Please follow [`SECURITY.md`](SECURITY.md). Do not disclose exploitable details in public issues.
