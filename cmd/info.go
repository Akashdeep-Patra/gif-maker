// cmd/info.go
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [video file]",
	Short: "Display information about a video file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		videoPath := args[0]

		// Check if the file exists
		if _, err := os.Stat(videoPath); os.IsNotExist(err) {
			return fmt.Errorf("video file does not exist: %s", videoPath)
		}

		// Get video information
		info, err := GetVideoInfo(videoPath)
		if err != nil {
			return fmt.Errorf("failed to get video information: %w", err)
		}

		// Get file size
		stat, err := os.Stat(videoPath)
		if err != nil {
			return fmt.Errorf("failed to get file size: %w", err)
		}

		// Display information
		color.Green("Video Information: %s", videoPath)
		fmt.Println("")

		fmt.Printf("Size:      %s\n", HumanizeBytes(stat.Size()))

		if width, ok := info["width"]; ok {
			fmt.Printf("Width:     %s px\n", width)
		}

		if height, ok := info["height"]; ok {
			fmt.Printf("Height:    %s px\n", height)
		}

		if duration, ok := info["duration"]; ok {
			durationFloat, err := strconv.ParseFloat(duration, 64)
			if err == nil {
				minutes := int(durationFloat) / 60
				seconds := int(durationFloat) % 60
				fmt.Printf("Duration:  %d:%02d (%.2f seconds)\n", minutes, seconds, durationFloat)
			} else {
				fmt.Printf("Duration:  %s seconds\n", duration)
			}
		}

		if frameRate, ok := info["r_frame_rate"]; ok {
			// Frame rate can be in the format "30000/1001" (for 29.97 fps)
			if strings.Contains(frameRate, "/") {
				parts := strings.Split(frameRate, "/")
				if len(parts) == 2 {
					num, err1 := strconv.ParseFloat(parts[0], 64)
					den, err2 := strconv.ParseFloat(parts[1], 64)
					if err1 == nil && err2 == nil && den > 0 {
						fps := num / den
						fmt.Printf("FPS:       %.2f\n", fps)
					} else {
						fmt.Printf("FPS:       %s\n", frameRate)
					}
				}
			} else {
				fmt.Printf("FPS:       %s\n", frameRate)
			}
		}

		// Calculate estimated GIF sizes
		if width, ok := info["width"]; ok {
			if height, ok2 := info["height"]; ok2 {
				if duration, ok3 := info["duration"]; ok3 {
					w, _ := strconv.Atoi(width)
					h, _ := strconv.Atoi(height)
					d, _ := strconv.ParseFloat(duration, 64)

					// Rough estimation for different FPS values
					fmt.Println("\nEstimated GIF sizes (rough approximation):")
					for _, fps := range []int{5, 10, 15, 20} {
						// Very rough approximation: pixels * frames * bytes per pixel / compression factor
						frames := int(d) * fps
						sizeBytes := float64(w*h*frames*3) / 4.0 // Assuming some compression
						fmt.Printf("  At %d FPS: ~%s\n", fps, HumanizeBytes(int64(sizeBytes)))
					}
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
