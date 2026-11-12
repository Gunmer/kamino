GO ?= go
BIN ?= bin/kamino
VERSION ?= 0.8.0
LDFLAGS = -X github.com/Gunmer/kamino/internal/version.Version=$(VERSION) -X github.com/Gunmer/kamino/internal/buildinfo.SelfContained=true

.PHONY: build test lint fmt bench-startup golden guard sync-catalog release-snapshot clean

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
	$(GO) test ./internal/performance -run Startup -v

golden:
	@if [ -x legacy/python/.venv/bin/python ]; then \
		legacy/python/.venv/bin/python scripts/gen_golden.py; \
	elif [ -x .venv/bin/python ]; then \
		.venv/bin/python scripts/gen_golden.py; \
	else \
		echo "se requiere un entorno Python (legacy/python/.venv) para generar los fixtures dorados"; \
		exit 1; \
	fi

guard:
	scripts/guard.sh

sync-catalog:
	cp -R src/kamino/catalog/apps/. internal/catalog/apps/
	scripts/check_catalog_sync.sh

release-snapshot:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "-s -w $(LDFLAGS)" -o dist/kamino-arm64 ./cmd/kamino
	./dist/kamino-arm64 --version

clean:
	rm -rf bin dist
