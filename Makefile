VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/oscarhugopaz/earth-cli/internal/version.Version=$(VERSION) \
	-X github.com/oscarhugopaz/earth-cli/internal/version.Commit=$(COMMIT) \
	-X github.com/oscarhugopaz/earth-cli/internal/version.Date=$(DATE)

.PHONY: build test vet fmt check snapshot clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/earth ./cmd/earth

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check: test vet build

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin dist
