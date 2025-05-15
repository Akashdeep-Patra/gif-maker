// cmd/util.go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CheckFFmpeg checks if FFmpeg is installed and returns an error if not
func CheckFFmpeg() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("FFmpeg not found in PATH: %w", err)
	}
	return nil
}

// GetVideoInfo uses FFmpeg to extract basic information about a video file
func GetVideoInfo(videoPath string) (map[string]string, error) {
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("video file does not exist: %s", videoPath)
	}

	// Run ffprobe to get video info
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,duration,r_frame_rate",
		"-of", "default=noprint_wrappers=1",
		videoPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	// Parse the output
	lines := strings.Split(string(output), "\n")
	info := make(map[string]string)

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "=")
		if len(parts) == 2 {
			info[parts[0]] = parts[1]
		}
	}

	return info, nil
}

// GetOptimalThreads returns the optimal number of threads to use based on CPU cores
func GetOptimalThreads() int {
	numCPU := runtime.NumCPU()
	if numCPU <= 2 {
		return 1
	} else if numCPU <= 4 {
		return 2
	} else {
		return numCPU - 2 // Leave some cores for other processes
	}
}

// HumanizeBytes converts bytes to a human-readable format (KB, MB, GB)
func HumanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ValidateTimeFormat checks if a time string is in the format HH:MM:SS or HH:MM:SS.MS
func ValidateTimeFormat(timeStr string) bool {
	if timeStr == "" {
		return true
	}

	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return false
	}

	// Check if each part is a valid number
	for i, part := range parts {
		// For seconds, we might have decimal points
		if i == 2 && strings.Contains(part, ".") {
			secParts := strings.Split(part, ".")
			if len(secParts) != 2 {
				return false
			}

			// Check if both parts of seconds are valid numbers
			if _, err := fmt.Sscanf(secParts[0], "%d", new(int)); err != nil {
				return false
			}

			if _, err := fmt.Sscanf(secParts[1], "%d", new(int)); err != nil {
				return false
			}
		} else {
			if _, err := fmt.Sscanf(part, "%d", new(int)); err != nil {
				return false
			}
		}
	}

	return true
}
