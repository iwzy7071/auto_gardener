.PHONY: test vet js-check build windows-build check package-windows package-macos clean

test:
	go test ./...

vet:
	go vet ./...

js-check:
	node --check web/static/app.js

build:
	go build -o bin/auto_gardener_server ./cmd/server

windows-build:
	GOOS=windows GOARCH=amd64 go build -o bin/gardener.exe ./cmd/server

check: test vet js-check windows-build

package-windows:
	./scripts/build-windows-package.sh

package-macos:
	./scripts/build-macos-package.sh

clean:
	rm -rf bin dist dist-test
