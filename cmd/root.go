// cmd/root.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	logger  *logrus.Logger
)

var rootCmd = &cobra.Command{
	Use:   "gif-maker",
	Short: "Convert videos to GIFs with customizable options",
	Long: `GIF Maker - A production-grade CLI tool for converting video files to GIFs.
	
Features:
- Interactive mode for easy use
- Customizable quality, size, and frame rate
- Simple command-line interface
- Progress tracking and logging`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		setupLogging()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	logger = logrus.New()
}

func setupLogging() {
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set up log file
	logDir := filepath.Join(os.TempDir(), "gif-maker-logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Warning: Could not create log directory: %v\n", err)
		return
	}

	logFile := filepath.Join(logDir, "gif-maker.log")
	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Warning: Could not set up log file: %v\n", err)
		return
	}

	logger.SetOutput(f)
	logger.Info("GIF Maker started")
}

func GetLogger() *logrus.Logger {
	return logger
}
