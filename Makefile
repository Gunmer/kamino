GO ?= go
BIN ?= bin/kamino
VERSION ?= $(shell sed -n 's/^var Version = "\(.*\)"$$/\1/p' internal/version/version.go)
LDFLAGS = -X github.com/Gunmer/kamino/internal/version.Version=$(VERSION) -X github.com/Gunmer/kamino/internal/buildinfo.SelfContained=true

.PHONY: build test lint fmt bench-startup release-snapshot clean

build:
	$(GO) build -trimpath -o $(BIN) ./cmd/kamino

test:
	$(GO) test ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint no disponible; se usa go vet"; \
		$(GO) vet ./...; \
	fi

fmt:
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
	else \
		echo "gofumpt no disponible; se usa gofmt"; \
		gofmt -w .; \
	fi

bench-startup:
	$(GO) test ./cmd/kamino -run Startup -v

release-snapshot:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "-s -w $(LDFLAGS)" -o dist/kamino-arm64 ./cmd/kamino
	./dist/kamino-arm64 --version

clean:
	rm -rf bin dist
