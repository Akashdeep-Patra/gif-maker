// cmd/version.go
package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Run: func(cmd *cobra.Command, args []string) {
		// Print application version
		color.Green("GIF Maker v1.0.0")
		fmt.Println("A command-line tool to convert videos to GIFs")
		fmt.Println("Source: https://github.com/akashdeep/gif-maker")
		fmt.Println("")

		// Check for FFmpeg installation
		_, err := exec.LookPath("ffmpeg")
		if err != nil {
			color.Red("❌ FFmpeg not found in PATH!")
			fmt.Println("This tool requires FFmpeg to work. Please install it:")
			fmt.Println("- MacOS: brew install ffmpeg")
			fmt.Println("- Ubuntu/Debian: sudo apt install ffmpeg")
			fmt.Println("- Windows: https://ffmpeg.org/download.html")
			return
		}

		// Get FFmpeg version
		ffmpegCmd := exec.Command("ffmpeg", "-version")
		output, err := ffmpegCmd.Output()
		if err != nil {
			color.Yellow("⚠️ FFmpeg found but unable to determine version")
		} else {
			outputStr := string(output)
			lines := strings.Split(outputStr, "\n")
			if len(lines) > 0 {
				color.Green("✅ %s", lines[0])
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
