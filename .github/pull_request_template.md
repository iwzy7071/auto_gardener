## Summary

-

## Testing

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `node --check web/static/app.js`
- [ ] `GOOS=windows GOARCH=amd64 go build -o /tmp/gardener.exe ./cmd/server`

## Safety checklist

- [ ] No passwords, tokens, private keys, server addresses, generated `frpc.toml`, `htpasswd`, runtime data, or packaged binaries are included.
- [ ] Windows support is preserved.
- [ ] User-facing changes are documented.
