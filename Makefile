.PHONY: test vet js-check build windows-build linux-build check package-windows package-macos package-linux clean

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

linux-build:
	GOOS=linux GOARCH=amd64 go build -o bin/gardener-linux-amd64 ./cmd/server

check: test vet js-check windows-build linux-build

package-windows:
	./scripts/build-windows-package.sh

package-macos:
	./scripts/build-macos-package.sh

package-linux:
	./scripts/build-linux-package.sh

clean:
	rm -rf bin dist dist-test
