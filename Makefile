.PHONY: all build install run test test-coverage clean lint release-check release-local build-all
.PHONY: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64

VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

all: build

build:
	go build $(LDFLAGS)

install:
	go install $(LDFLAGS)

run: build
	./pj

test:
	go test -cover ./...

test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run

# Cross-compilation targets
build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/pj-linux-amd64

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/pj-linux-arm64

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/pj-darwin-amd64

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/pj-darwin-arm64

build-windows-amd64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/pj-windows-amd64.exe

build-windows-arm64:
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/pj-windows-arm64.exe

build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64
	@echo "All platform builds completed in dist/"

# GoReleaser targets
release-check:
	goreleaser check

release-local:
	goreleaser release --snapshot --clean --skip=publish

clean:
	rm -f pj
	rm -rf dist/
