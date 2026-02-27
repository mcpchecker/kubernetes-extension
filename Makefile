BINARY_NAME := kubernetes-extension
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.PHONY: build
build:
	go build -o bin/$(BINARY_NAME) ./cmd

.PHONY: test
test:
	go test -v -race ./...

.PHONY: clean
clean:
	rm -rf bin dist

# Release targets for CI/CD
.PHONY: build-release
build-release:
	@echo "Building release binary for $(GOOS)/$(GOARCH)..."
	@mkdir -p dist
	@if [ "$(GOOS)" = "windows" ]; then \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags="-s -w" -o "dist/$(BINARY_NAME)-$(GOOS)-$(GOARCH).exe" ./cmd; \
	else \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags="-s -w" -o "dist/$(BINARY_NAME)-$(GOOS)-$(GOARCH)" ./cmd; \
	fi
	@echo "Build complete!"

# Version validation targets
.PHONY: validate-release-tag
validate-release-tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make validate-release-tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Validating release tag format: $(VERSION)"
	@if echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "✓ Release tag $(VERSION) is valid"; \
	else \
		echo "✗ Error: Release tag must match format 'vX.Y.Z' (no suffixes)"; \
		echo "  Got: $(VERSION)"; \
		exit 1; \
	fi

.PHONY: validate-prerelease-tag
validate-prerelease-tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make validate-prerelease-tag VERSION=v1.0.0-rc.1"; \
		exit 1; \
	fi
	@echo "Validating prerelease tag format: $(VERSION)"
	@if echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$$'; then \
		echo "✓ Prerelease tag $(VERSION) is valid"; \
	else \
		echo "✗ Error: Prerelease tag must match format 'vX.Y.Z-rc.N'"; \
		echo "  Got: $(VERSION)"; \
		exit 1; \
	fi
