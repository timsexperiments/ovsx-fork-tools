.PHONY: build test e2e-local e2e-docker clean

build:
	go build -o ovsx-setup main.go

test:
	go test -v ./...

e2e: build
	go test -v -tags e2e ./tests/e2e/...

clean:
	rm -f ovsx-setup
