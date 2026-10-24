VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/oscarhugopaz/earth-cli/internal/version.Version=$(VERSION) \
	-X github.com/oscarhugopaz/earth-cli/internal/version.Commit=$(COMMIT) \
	-X github.com/oscarhugopaz/earth-cli/internal/version.Date=$(DATE)

.PHONY: build test test-integration vet fmt vuln check snapshot clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/earth ./cmd/earth

test:
	go test ./...

test-integration:
	EARTH_INTEGRATION=1 go test -count=1 -run Integration -timeout 300s ./internal/providers/copernicus/

vet:
	go vet ./...

fmt:
	gofmt -w .

# Checks the module and its dependencies against the Go vulnerability database,
# including the standard library of the toolchain running the scan.
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...

check: test vet build

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin dist
