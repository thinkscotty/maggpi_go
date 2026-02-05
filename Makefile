# MaggPi Makefile

BINARY_NAME=maggpi
BUILD_DIR=bin
INSTALL_DIR=/opt/maggpi

# Build for current platform
.PHONY: build
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/maggpi

# Build for Raspberry Pi (Linux ARM64)
.PHONY: build-pi
build-pi:
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/maggpi

# Build for Linux AMD64
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/maggpi

# Run locally
.PHONY: run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run with hot reload (requires air: go install github.com/air-verse/air@latest)
.PHONY: dev
dev:
	air

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	rm -rf data/*.db

# Download dependencies
.PHONY: deps
deps:
	go mod tidy
	go mod download

# Run tests
.PHONY: test
test:
	go test -v ./...

# Create release package for Raspberry Pi
.PHONY: release-pi
release-pi: build-pi
	mkdir -p release
	tar -czvf release/maggpi-linux-arm64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-linux-arm64 \
		-C .. web data

# Install on local system (for development)
.PHONY: install
install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	cp -r web $(INSTALL_DIR)/
	mkdir -p $(INSTALL_DIR)/data

# Help
.PHONY: help
help:
	@echo "MaggPi Build Commands:"
	@echo "  make build      - Build for current platform"
	@echo "  make build-pi   - Build for Raspberry Pi (Linux ARM64)"
	@echo "  make build-linux- Build for Linux AMD64"
	@echo "  make run        - Build and run locally"
	@echo "  make dev        - Run with hot reload (requires air)"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make deps       - Download dependencies"
	@echo "  make test       - Run tests"
	@echo "  make release-pi - Create release package for Pi"
	@echo "  make install    - Install to /opt/maggpi"
