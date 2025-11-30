.PHONY: build test e2e-local e2e-docker clean

build:
	go build -o ovsx-setup main.go

test:
	go test -v ./...

e2e-local: build
	E2E=true go test -v ./tests/e2e/...

e2e-docker:
	./scripts/run-e2e-docker.sh

clean:
	rm -f ovsx-setup
	rm -rf tests/e2e/fixtures/extension/pnpm-lock.yaml
