# Makefile for GIF-Maker

# Variables
BINARY_NAME=gif-maker
MAIN_PACKAGE=.
GO=go
GOFMT=gofmt -s -w
GOVET=$(GO) vet
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=./build
LDFLAGS=-ldflags "-X gif-maker/cmd.Version=$(VERSION)"
VIDEO_MOV=video.mov
FFMPEG_DIR=internal/ffmpeg/binaries

# Colors for terminal output
BLUE=\033[0;34m
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all build clean test lint fmt vet install run help check release test-run run-new-build run-convert examples download-ffmpeg

# Default target
all: check build

# Build the application
build: download-ffmpeg-check
	@echo "${BLUE}Building $(BINARY_NAME)...${NC}"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "${GREEN}Build successful! Binary: $(BUILD_DIR)/$(BINARY_NAME)${NC}"

# Download FFmpeg binaries for embedding
download-ffmpeg:
	@echo "${BLUE}Downloading FFmpeg binaries for packaging...${NC}"
	@mkdir -p $(FFMPEG_DIR)
	@chmod +x scripts/download_ffmpeg.sh
	@./scripts/download_ffmpeg.sh
	@echo "${GREEN}FFmpeg binaries downloaded for packaging${NC}"

# Check if FFmpeg binaries are already downloaded
download-ffmpeg-check:
	@if [ -z "$$(find $(FFMPEG_DIR) -type f -name "ffmpeg-*" -print -quit)" ]; then \
		echo "${YELLOW}No embedded FFmpeg binaries found. Downloading now...${NC}"; \
		$(MAKE) download-ffmpeg; \
	else \
		echo "${GREEN}Found embedded FFmpeg binaries.${NC}"; \
	fi

# Clean build artifacts
clean:
	@echo "${BLUE}Cleaning build artifacts...${NC}"
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "${GREEN}Clean complete${NC}"

# Clean FFmpeg binaries
clean-ffmpeg:
	@echo "${BLUE}Cleaning FFmpeg binaries...${NC}"
	@rm -rf $(FFMPEG_DIR)/*
	@mkdir -p $(FFMPEG_DIR)
	@echo "${GREEN}FFmpeg binaries cleaned${NC}"

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

# Build and run conversion with video.mov file
run-new-build: clean build check
	@echo "${BLUE}=== Starting fresh conversion with a new build ===${NC}"
	@echo "${YELLOW}✓ Cleaned previous builds${NC}"
	@echo "${YELLOW}✓ Created new build${NC}"
	@echo "${YELLOW}✓ Checked dependencies${NC}"
	@echo "${BLUE}=== Starting conversion process ===${NC}"
	@if [ -f "$(VIDEO_MOV)" ]; then \
		echo "${GREEN}Converting $(VIDEO_MOV) to GIF...${NC}"; \
		echo "${YELLOW}Using settings: 15 fps, 95% quality${NC}"; \
		FORCE_COLOR=1 stdbuf -o0 $(BUILD_DIR)/$(BINARY_NAME) convert -i "$(VIDEO_MOV)" -o output.gif -f 6 -q 20; \
		if [ $$? -eq 0 ]; then \
			echo "${GREEN}=== Conversion completed successfully ===${NC}"; \
			echo "${YELLOW}Output saved to: output.gif${NC}"; \
		else \
			echo "${RED}=== Conversion failed ===${NC}"; \
		fi \
	else \
		echo "${RED}Error: $(VIDEO_MOV) not found in current directory.${NC}"; \
		echo "${YELLOW}Please add a video.mov file or specify a different file by updating the VIDEO_MOV variable:${NC}"; \
		echo "make run-new-build VIDEO_MOV=path/to/your/video.mov"; \
	fi

# Run with real-time progress display
run-convert: build
	@echo "${BLUE}Running conversion with real-time progress...${NC}"
	@echo "${YELLOW}Note: This command ensures progress bar updates in real-time${NC}"
	@if [ -n "$$(find . -type f -name "*.mp4" -o -name "*.mov" -o -name "*.avi" -print -quit)" ]; then \
		VIDEO_FILE=$$(find . -type f -name "*.mp4" -o -name "*.mov" -o -name "*.avi" -print -quit); \
		echo "${GREEN}Converting $${VIDEO_FILE}...${NC}"; \
		FORCE_COLOR=1 stdbuf -o0 $(BUILD_DIR)/$(BINARY_NAME) convert -i "$${VIDEO_FILE}" -o output.gif -v; \
	else \
		echo "${RED}No video files found. Please add a video file to the current directory.${NC}"; \
	fi

# Check dependencies
check:
	@echo "${BLUE}Checking dependencies...${NC}"
	@# Check if FFmpeg is either embedded or in system path
	@if [ -n "$$(find $(FFMPEG_DIR) -type f -name "ffmpeg-*" -print -quit)" ]; then \
		echo "${GREEN}Found embedded FFmpeg binaries.${NC}"; \
	elif which ffmpeg > /dev/null; then \
		echo "${GREEN}Found FFmpeg in system path.${NC}"; \
	else \
		echo "${YELLOW}No FFmpeg found. Will download during build.${NC}"; \
	fi
	@echo "${GREEN}All dependencies satisfied${NC}"

# Create a release package with embedded FFmpeg
release: download-ffmpeg build
	@echo "${BLUE}Creating release package...${NC}"
	@mkdir -p $(BUILD_DIR)/release
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/release/
	@cp README.md $(BUILD_DIR)/release/
	@cp LICENSE $(BUILD_DIR)/release/ 2>/dev/null || echo "No LICENSE file found"
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION).tar.gz release
	@echo "${GREEN}Release package created: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION).tar.gz${NC}"

# Cross-compile for multiple platforms with embedded FFmpeg
cross-compile: clean download-ffmpeg
	@echo "${BLUE}Cross-compiling for multiple platforms...${NC}"
	@mkdir -p $(BUILD_DIR)/release

	# Linux (amd64)
	@echo "${YELLOW}Building for Linux (amd64)...${NC}"
	@GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	
	# macOS (amd64)
	@echo "${YELLOW}Building for macOS (amd64)...${NC}"
	@GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	
	# macOS (arm64)
	@echo "${YELLOW}Building for macOS (arm64)...${NC}"
	@GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	
	# Windows (amd64)
	@echo "${YELLOW}Building for Windows (amd64)...${NC}"
	@GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	
	@echo "${GREEN}Cross-compilation complete. Binaries available in $(BUILD_DIR)/release/${NC}"

# Test basic functionality without conversion
test-run: build
	@echo "${BLUE}Testing basic functionality...${NC}"
	@echo "${YELLOW}Testing version command:${NC}"
	@$(BUILD_DIR)/$(BINARY_NAME) version
	@echo "\n${YELLOW}Testing help command:${NC}"
	@$(BUILD_DIR)/$(BINARY_NAME) --help

# Examples of usage with robust error handling
examples: build check
	@echo "${BLUE}Running examples...${NC}"
	
	@echo "${YELLOW}Example: Show help for convert command${NC}"
	@$(BUILD_DIR)/$(BINARY_NAME) convert --help
	
	@# Find an example video file if available
	@EXAMPLE_VIDEO=$$(find . -type f -name "*.mp4" -o -name "*.mov" -o -name "*.avi" -print -quit); \
	if [ -n "$$EXAMPLE_VIDEO" ]; then \
		echo "\n${YELLOW}Example: Show video info for $$EXAMPLE_VIDEO${NC}"; \
		$(BUILD_DIR)/$(BINARY_NAME) info "$$EXAMPLE_VIDEO"; \
		echo "\n${YELLOW}Example: Convert video to GIF${NC}"; \
		FORCE_COLOR=1 stdbuf -o0 $(BUILD_DIR)/$(BINARY_NAME) convert -i "$$EXAMPLE_VIDEO" -o output.gif --width 640 -v || \
		echo "\n${RED}Conversion failed. Please check if FFmpeg is installed correctly.${NC}"; \
	else \
		echo "\n${YELLOW}No example video found. Showing sample commands:${NC}"; \
		echo "${GREEN}To show video info:${NC}"; \
		echo "  $(BUILD_DIR)/$(BINARY_NAME) info YOUR_VIDEO_FILE.mp4"; \
		echo "${GREEN}To convert a video to GIF:${NC}"; \
		echo "  $(BUILD_DIR)/$(BINARY_NAME) convert -i YOUR_VIDEO_FILE.mp4 -o output.gif"; \
		echo "${YELLOW}Important:${NC} Always provide a file path immediately after the -i flag"; \
		echo "\n${GREEN}To use interactive mode:${NC}"; \
		echo "  $(BUILD_DIR)/$(BINARY_NAME) convert --interactive"; \
	fi
	
	@echo "\n${GREEN}Examples completed.${NC}"

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
	@echo "  ${GREEN}make run-new-build${NC} - Build and convert video.mov to GIF"
	@echo "  ${GREEN}make run-convert${NC}   - Convert a video with real-time progress display"
	@echo "  ${GREEN}make check${NC}         - Check dependencies"
	@echo "  ${GREEN}make release${NC}       - Create a release package"
	@echo "  ${GREEN}make cross-compile${NC} - Build for multiple platforms"
	@echo "  ${GREEN}make test-run${NC}      - Test basic functionality without conversion"
	@echo "  ${GREEN}make examples${NC}      - Run example commands"
	@echo "  ${GREEN}make help${NC}          - Show this help message"
	@echo "\n${YELLOW}Command usage notes:${NC}"
	@echo "  - Always provide values immediately after flags (e.g., -i input.mp4, not just -i)"
	@echo "  - For interactive mode: ${GREEN}$(BUILD_DIR)/$(BINARY_NAME) convert --interactive${NC}"
	@echo "  - To specify a different video for run-new-build: ${GREEN}make run-new-build VIDEO_MOV=path/to/video.mov${NC}"

# Default to help if no target is specified
.DEFAULT_GOAL := help 