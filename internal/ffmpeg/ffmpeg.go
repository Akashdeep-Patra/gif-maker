// Package ffmpeg provides functionality to manage and use embedded FFmpeg binaries.
package ffmpeg

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

//go:embed binaries/*
var embeddedBinaries embed.FS

// Manager handles the extraction and usage of embedded FFmpeg binaries
type Manager struct {
	binariesDir     string
	extractedPath   string
	extractedBinary string
	mu              sync.Mutex
	extracted       bool
}

// NewManager creates a new FFmpeg manager
func NewManager() *Manager {
	return &Manager{
		binariesDir: "binaries",
		extracted:   false,
	}
}

// GetPath returns the path to the FFmpeg binary
func (m *Manager) GetPath() (string, error) {
	// Check if we've already extracted the binary
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.extracted && m.extractedBinary != "" {
		// Verify the extracted binary still exists
		if _, err := os.Stat(m.extractedBinary); err == nil {
			return m.extractedBinary, nil
		}
		// If it doesn't exist, fall through to extract it again
		m.extracted = false
	}

	// Extract the binary if needed
	return m.extractBinary()
}

// extractBinary extracts the appropriate FFmpeg binary for the current platform
// Must be called with the mutex held
func (m *Manager) extractBinary() (string, error) {
	// Determine the binary name based on OS
	binaryName := getBinaryNameForPlatform()
	if binaryName == "" {
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Create a temporary directory for the extracted binary
	tempDir, err := os.MkdirTemp("", "ffmpeg-extract")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Save the extraction path for cleanup later
	m.extractedPath = tempDir

	// Construct the embedded path and extract the binary
	embeddedPath := filepath.Join(m.binariesDir, binaryName)

	// Read the embedded binary
	binaryData, err := embeddedBinaries.ReadFile(embeddedPath)
	if err != nil {
		// If the embedded binary isn't found, check for system installation
		return m.findSystemFFmpeg()
	}

	// Determine the output path
	outputPath := filepath.Join(tempDir, binaryName)
	if runtime.GOOS != "windows" {
		// On non-Windows platforms, don't include the extension
		outputPath = filepath.Join(tempDir, "ffmpeg")
	}

	// Write the binary to the temp directory
	if err := os.WriteFile(outputPath, binaryData, 0755); err != nil {
		return "", fmt.Errorf("failed to extract FFmpeg: %w", err)
	}

	// Save the path and mark as extracted
	m.extractedBinary = outputPath
	m.extracted = true

	return outputPath, nil
}

// findSystemFFmpeg attempts to find a system-installed FFmpeg binary
func (m *Manager) findSystemFFmpeg() (string, error) {
	// Check if ffmpeg is available in PATH
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("FFmpeg not found in embedded binaries or system PATH")
	}

	// Use the system ffmpeg
	m.extractedBinary = path
	m.extracted = true

	return path, nil
}

// Cleanup removes the extracted files
func (m *Manager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.extractedPath != "" {
		if err := os.RemoveAll(m.extractedPath); err != nil {
			return fmt.Errorf("failed to clean up extracted files: %w", err)
		}
		m.extractedPath = ""
		m.extractedBinary = ""
		m.extracted = false
	}

	return nil
}

// getBinaryNameForPlatform returns the FFmpeg binary filename for the current platform
func getBinaryNameForPlatform() string {
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			return "ffmpeg-win64.exe"
		case "386":
			return "ffmpeg-win32.exe"
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return "ffmpeg-macos-x86_64"
		case "arm64":
			return "ffmpeg-macos-arm64"
		}
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "ffmpeg-linux-x86_64"
		case "386":
			return "ffmpeg-linux-i386"
		case "arm64":
			return "ffmpeg-linux-arm64"
		case "arm":
			return "ffmpeg-linux-armhf"
		}
	}
	return ""
}
