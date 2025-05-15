# Makefile for GIF-Maker

# Variables
BINARY_NAME=gif-maker
MAIN_PACKAGE=.
GO=go
GOFMT=gofmt -s -w
GOVET=$(GO) vet
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=./build
LDFLAGS=-ldflags "-X github.com/Akashdeep-Patra/gif-maker/cmd.Version=$(VERSION)"
FFMPEG_DIR=internal/ffmpeg/binaries
TEST_VIDEO=video.mp4
TEST_SCRIPT=scripts/test-with-video.sh
INVALID_TEST_SCRIPT=scripts/test-with-invalid-file.sh

# Colors for terminal output
BLUE=\033[0;34m
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all build clean test lint fmt vet install check help build-and-test

# Default target
all: check build

# Build the application
build: 
	@echo "${BLUE}Building $(BINARY_NAME)...${NC}"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "${GREEN}Build successful! Binary: $(BUILD_DIR)/$(BINARY_NAME)${NC}"

# Clean build artifacts
clean:
	@echo "${BLUE}Cleaning build artifacts...${NC}"
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "${GREEN}Clean complete${NC}"

# Run tests
test:
	@echo "${BLUE}Running tests...${NC}"
	$(GO) test -v ./...

# Run linting tools
lint: fmt vet
	@echo "${GREEN}Linting complete${NC}"

# Format source code
fmt:
	@echo "${BLUE}Formatting source code...${NC}"
	$(GOFMT) .

# Run go vet
vet:
	@echo "${BLUE}Running go vet...${NC}"
	$(GOVET) ./...

# Install the application
install: build
	@echo "${BLUE}Installing $(BINARY_NAME)...${NC}"
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "${GREEN}Installation complete. Run '$(BINARY_NAME)' to use.${NC}"

# Install to local system (requires root/sudo)
install-system: build
	@echo "${BLUE}Installing $(BINARY_NAME) to system...${NC}"
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "${GREEN}Installation complete. Run '$(BINARY_NAME)' to use.${NC}"

# Run the application
run:
	@echo "${BLUE}Running $(BINARY_NAME)...${NC}"
	$(GO) run $(MAIN_PACKAGE)

# Check dependencies
check:
	@echo "${BLUE}Checking dependencies...${NC}"
	@if which ffmpeg > /dev/null; then \
		echo "${GREEN}Found FFmpeg in system path.${NC}"; \
	else \
		echo "${YELLOW}No FFmpeg found in system path. Please install FFmpeg for optimal performance.${NC}"; \
	fi
	@echo "${GREEN}All dependencies checked${NC}"

# Create a release (for testing)
release: build
	@echo "${BLUE}Creating release package...${NC}"
	@mkdir -p $(BUILD_DIR)/release
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/release/
	@cp README.md $(BUILD_DIR)/release/
	@cp LICENSE $(BUILD_DIR)/release/ 2>/dev/null || echo "No LICENSE file found"
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION).tar.gz release
	@echo "${GREEN}Release package created: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION).tar.gz${NC}"

# Build and test with valid and invalid files
build-and-test: build
	@echo "${BLUE}Running all tests for $(BINARY_NAME)...${NC}"
	@echo "${BLUE}=== Testing with valid video file ===${NC}"
	@./$(TEST_SCRIPT) $(TEST_VIDEO) || { echo "${RED}Video test failed${NC}"; exit 1; }
	@echo "${BLUE}=== Testing with invalid file format ===${NC}"
	@./$(INVALID_TEST_SCRIPT) || { echo "${RED}Invalid file test failed${NC}"; exit 1; }
	@echo "${GREEN}\033[1mAll tests passed!${NC}"

# Help target
help:
	@echo "${BLUE}GIF-Maker Makefile Help${NC}"
	@echo "${YELLOW}Available commands:${NC}"
	@echo "  ${GREEN}make${NC}               - Check dependencies and build the application"
	@echo "  ${GREEN}make build${NC}         - Build the application"
	@echo "  ${GREEN}make clean${NC}         - Remove build artifacts"
	@echo "  ${GREEN}make test${NC}          - Run tests"
	@echo "  ${GREEN}make lint${NC}          - Run formatting and static analysis"
	@echo "  ${GREEN}make install${NC}       - Install to GOPATH/bin"
	@echo "  ${GREEN}make install-system${NC} - Install to /usr/local/bin (may require sudo)"
	@echo "  ${GREEN}make run${NC}           - Run the application"
	@echo "  ${GREEN}make check${NC}         - Check dependencies"
	@echo "  ${GREEN}make release${NC}       - Create a basic release package"
	@echo "  ${GREEN}make build-and-test${NC} - Build and test with both valid and invalid files"
	@echo "  ${GREEN}make help${NC}          - Show this help message"

# Default to help if no target is specified
.DEFAULT_GOAL := help 