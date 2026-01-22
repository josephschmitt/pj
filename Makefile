.PHONY: all build install run test clean

all: build

build:
	go build -ldflags "-X main.version=$$(git describe --tags --abbrev=0 2>/dev/null || echo dev)"

install:
	go install -ldflags "-X main.version=$$(git describe --tags --abbrev=0 2>/dev/null || echo dev)"

run: build
	./pj

test:
	go test -cover ./...

clean:
	rm -f pj
