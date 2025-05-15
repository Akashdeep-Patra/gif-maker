// cmd/convert.go
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Akashdeep-Patra/gif-maker/internal/ffmpeg"
)

type ConvertOptions struct {
	Input       string
	Output      string
	FPS         int
	Start       string
	Duration    string
	Width       int
	Quality     int
	Interactive bool
	NoProgress  bool
}

var opts ConvertOptions

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert a video file to a GIF",
	Long: `Convert a video file to a GIF with customizable options.
You can either provide options via flags or use interactive mode.
If no arguments are provided, interactive mode is enabled by default.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Enable interactive mode automatically if no input is provided
		if opts.Input == "" && !opts.Interactive {
			// Check if any arguments or flags were specified
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				// No arguments or flags provided, default to interactive mode
				opts.Interactive = true
			} else {
				return fmt.Errorf("input file is required (use --input or -i)")
			}
		}

		// If interactive mode is enabled, prompt for values
		if opts.Interactive {
			if err := promptForOptions(); err != nil {
				return fmt.Errorf("error in interactive mode: %w", err)
			}
		}

		// Validate input file exists
		if _, err := os.Stat(opts.Input); os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", opts.Input)
		}

		// Set default output if not provided
		if opts.Output == "" {
			inputBase := filepath.Base(opts.Input)
			inputExt := filepath.Ext(inputBase)
			opts.Output = strings.TrimSuffix(inputBase, inputExt) + ".gif"
		}

		return convertVideo()
	},
}

// Add FFmpeg manager variable
var ffmpegManager *ffmpeg.Manager

// Update the init function to initialize the FFmpeg manager
func init() {
	convertCmd.Flags().StringVarP(&opts.Input, "input", "i", "", "Input video file (required unless using interactive mode)")
	convertCmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Output GIF file (default: input_name.gif)")
	convertCmd.Flags().IntVarP(&opts.FPS, "fps", "f", 10, "Frames per second")
	convertCmd.Flags().StringVar(&opts.Start, "start", "", "Start time (format: 00:00:00)")
	convertCmd.Flags().StringVar(&opts.Duration, "duration", "", "Duration (format: 00:00:00)")
	convertCmd.Flags().IntVarP(&opts.Width, "width", "w", 0, "Output width in pixels (default: same as input)")
	convertCmd.Flags().IntVarP(&opts.Quality, "quality", "q", 90, "Output quality (1-100)")
	convertCmd.Flags().BoolVarP(&opts.Interactive, "interactive", "I", false, "Use interactive mode (default if no arguments provided)")
	convertCmd.Flags().BoolVar(&opts.NoProgress, "no-progress", false, "Disable progress bar")

	// Initialize the FFmpeg manager
	ffmpegManager = ffmpeg.NewManager()

	rootCmd.AddCommand(convertCmd)
}

// Helper function to open a file explorer dialog
func openFileDialog(isInput bool) string {
	var cmd *exec.Cmd
	var output []byte
	var err error

	// Get a default filename for output based on current timestamp
	defaultGifName := fmt.Sprintf("output-%s.gif", time.Now().Format("20060102-150405"))

	// Determine the file dialog command based on OS
	switch runtime.GOOS {
	case "darwin":
		// macOS - use AppleScript to open a file dialog
		dialogType := "file"
		extraParams := ""
		promptText := "Select input video file"

		if !isInput {
			dialogType = "save"
			extraParams = fmt.Sprintf(`default name "%s"`, defaultGifName)
			promptText = "Save output GIF as"
		}

		// Create a temporary AppleScript file with improved default name handling
		scriptContent := fmt.Sprintf(`
			set theFile to choose %s with prompt "%s:" %s
			set thePath to POSIX path of theFile
			return thePath
		`, dialogType, promptText, extraParams)

		// Write script to temporary file
		tmpFile, err := os.CreateTemp("", "filepicker-*.scpt")
		if err == nil {
			defer os.Remove(tmpFile.Name())
			if _, err = tmpFile.WriteString(scriptContent); err == nil {
				tmpFile.Close()
				cmd = exec.Command("osascript", tmpFile.Name())
				output, err = cmd.Output()
			}
		}
	case "windows":
		// Windows - use PowerShell to open a file dialog
		dialogCode := ""
		if isInput {
			dialogCode = `[System.Reflection.Assembly]::LoadWithPartialName("System.windows.forms") | Out-Null
			$OpenFileDialog = New-Object System.Windows.Forms.OpenFileDialog
			$OpenFileDialog.Title = "Select a video file"
			$OpenFileDialog.filter = "Video files|*.mp4;*.avi;*.mov;*.mkv;*.webm|All files|*.*"
			$OpenFileDialog.ShowDialog() | Out-Null
			$OpenFileDialog.FileName`
		} else {
			dialogCode = `[System.Reflection.Assembly]::LoadWithPartialName("System.windows.forms") | Out-Null
			$SaveFileDialog = New-Object System.Windows.Forms.SaveFileDialog
			$SaveFileDialog.Title = "Save GIF as"
			$SaveFileDialog.filter = "GIF files|*.gif|All files|*.*"
			$SaveFileDialog.DefaultExt = "gif"
			$SaveFileDialog.FileName = "` + defaultGifName + `"
			$SaveFileDialog.ShowDialog() | Out-Null
			$SaveFileDialog.FileName`
		}
		cmd = exec.Command("powershell", "-Command", dialogCode)
		output, err = cmd.Output()
	case "linux":
		// Linux - use zenity if available
		dialogType := "--file-selection"
		dialogTitle := "Select input video file"
		extraParams := ""

		if !isInput {
			dialogType = "--file-selection --save"
			dialogTitle = "Save output GIF as"
			extraParams = fmt.Sprintf(`--filename="%s"`, defaultGifName)
		}

		args := []string{
			dialogType,
			"--title", dialogTitle,
		}

		if extraParams != "" {
			args = append(args, extraParams)
		}

		cmd = exec.Command("zenity", args...)
		output, err = cmd.Output()
	}

	// Process the output
	if err == nil && len(output) > 0 {
		path := strings.TrimSpace(string(output))

		// For output files, ensure GIF extension
		if !isInput && path != "" && !strings.HasSuffix(strings.ToLower(path), ".gif") {
			path += ".gif"
		}

		return path
	}

	return ""
}

func promptForOptions() error {
	// Ask if user wants to use file picker for input
	var useFilePicker bool
	pickerQuestion := &survey.Confirm{
		Message: "Would you like to use a file picker to select files?",
		Default: true,
	}
	if err := survey.AskOne(pickerQuestion, &useFilePicker); err != nil {
		return err
	}

	// Input file prompt
	if useFilePicker {
		fmt.Println("Opening file dialog, please select your input video file...")
		path := openFileDialog(true)
		if path != "" {
			opts.Input = path
			fmt.Printf("Selected file: %s\n", opts.Input)
		} else {
			// Fall back to text input if file dialog fails
			var inputQuestion = &survey.Input{
				Message: "Input video file path:",
				Help:    "Path to the video file you want to convert to a GIF",
			}
			if err := survey.AskOne(inputQuestion, &opts.Input, survey.WithValidator(survey.Required)); err != nil {
				return err
			}
		}
	} else {
		var inputQuestion = &survey.Input{
			Message: "Input video file path:",
			Help:    "Path to the video file you want to convert to a GIF",
		}
		if err := survey.AskOne(inputQuestion, &opts.Input, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	// Check if input file exists
	if _, err := os.Stat(opts.Input); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", opts.Input)
	}

	// Output file prompt
	defaultOutput := strings.TrimSuffix(opts.Input, filepath.Ext(opts.Input)) + ".gif"

	// Ask if user wants to use file picker for output
	if useFilePicker {
		fmt.Println("Opening file dialog, please choose where to save the output GIF...")
		path := openFileDialog(false)
		if path != "" {
			opts.Output = path
			// Ensure it has .gif extension
			if !strings.HasSuffix(strings.ToLower(opts.Output), ".gif") {
				opts.Output += ".gif"
			}
			fmt.Printf("Output will be saved to: %s\n", opts.Output)
		} else {
			// Fall back to text input
			var outputQuestion = &survey.Input{
				Message: "Output GIF file path:",
				Default: defaultOutput,
			}
			if err := survey.AskOne(outputQuestion, &opts.Output); err != nil {
				return err
			}
		}
	} else {
		// Ask if user wants to use file picker specifically for output
		// even if they didn't use it for input
		var useOutputFilePicker bool
		outputPickerQuestion := &survey.Confirm{
			Message: "Would you like to use a file picker to select the output location?",
			Default: true,
		}
		if err := survey.AskOne(outputPickerQuestion, &useOutputFilePicker); err != nil {
			return err
		}

		if useOutputFilePicker {
			fmt.Println("Opening file dialog, please choose where to save the output GIF...")
			path := openFileDialog(false)
			if path != "" {
				opts.Output = path
				// Ensure it has .gif extension
				if !strings.HasSuffix(strings.ToLower(opts.Output), ".gif") {
					opts.Output += ".gif"
				}
				fmt.Printf("Output will be saved to: %s\n", opts.Output)
			} else {
				// Fall back to text input if file dialog fails
				var outputQuestion = &survey.Input{
					Message: "Output GIF file path:",
					Default: defaultOutput,
				}
				if err := survey.AskOne(outputQuestion, &opts.Output); err != nil {
					return err
				}
			}
		} else {
			var outputQuestion = &survey.Input{
				Message: "Output GIF file path:",
				Default: defaultOutput,
			}
			if err := survey.AskOne(outputQuestion, &opts.Output); err != nil {
				return err
			}
		}
	}

	// FPS prompt
	var fpsQuestion = &survey.Input{
		Message: "Frames per second (higher = smoother but larger file):",
		Default: "10",
	}
	var fpsStr string
	if err := survey.AskOne(fpsQuestion, &fpsStr); err != nil {
		return err
	}
	fps, err := strconv.Atoi(fpsStr)
	if err != nil || fps < 1 {
		return fmt.Errorf("invalid FPS value: %s", fpsStr)
	}
	opts.FPS = fps

	// Start time prompt
	var startQuestion = &survey.Input{
		Message: "Start time (format: 00:00:00, leave empty for beginning):",
		Default: "",
	}
	if err := survey.AskOne(startQuestion, &opts.Start); err != nil {
		return err
	}

	// Duration prompt
	var durationQuestion = &survey.Input{
		Message: "Duration (format: 00:00:00, leave empty for full video):",
		Default: "",
	}
	if err := survey.AskOne(durationQuestion, &opts.Duration); err != nil {
		return err
	}

	// Width prompt
	var widthQuestion = &survey.Input{
		Message: "Width in pixels (leave empty to keep original size):",
		Default: "",
	}
	var widthStr string
	if err := survey.AskOne(widthQuestion, &widthStr); err != nil {
		return err
	}
	if widthStr != "" {
		width, err := strconv.Atoi(widthStr)
		if err != nil || width < 1 {
			return fmt.Errorf("invalid width value: %s", widthStr)
		}
		opts.Width = width
	}

	// Quality prompt
	var qualityOptions = []string{"Low (faster, smaller file)", "Medium", "High (slower, larger file)"}
	var qualityIndex int
	var qualityQuestion = &survey.Select{
		Message: "Select quality:",
		Options: qualityOptions,
		Default: 1,
	}
	if err := survey.AskOne(qualityQuestion, &qualityIndex); err != nil {
		return err
	}

	// Map quality selection to actual quality value
	switch qualityIndex {
	case 0:
		opts.Quality = 50
	case 1:
		opts.Quality = 75
	case 2:
		opts.Quality = 95
	}

	return nil
}

func convertVideo() error {
	logger := GetLogger()
	logger.Infof("Starting conversion: %s -> %s", opts.Input, opts.Output)

	// Check if FFmpeg is installed
	if err := checkFFmpegInstallation(); err != nil {
		return err
	}

	// Get FFmpeg path from the manager
	ffmpegPath, err := ffmpegManager.GetPath()
	if err != nil {
		return fmt.Errorf("Failed to get FFmpeg: %w", err)
	}

	// Prepare FFmpeg arguments
	ffmpegArgs := []string{"-i", opts.Input}

	// Add global options for better compatibility
	ffmpegArgs = append([]string{
		"-y",
		"-loglevel", "info",
		"-threads", fmt.Sprintf("%d", GetOptimalThreads()),
		"-progress", "pipe:1",
		"-stats_period", "0.1",
	}, ffmpegArgs...)

	if opts.Start != "" {
		ffmpegArgs = append(ffmpegArgs, "-ss", opts.Start)
	}

	if opts.Duration != "" {
		ffmpegArgs = append(ffmpegArgs, "-t", opts.Duration)
	}

	// Build the filter string
	filterComplex := fmt.Sprintf("fps=%d", opts.FPS)

	if opts.Width > 0 {
		filterComplex = fmt.Sprintf("%s,scale=%d:-1:flags=lanczos", filterComplex, opts.Width)
	}

	// Add the quality parameter (using palettegen for better quality)
	filterComplex = fmt.Sprintf("%s,split[s0][s1];[s0]palettegen=max_colors=256:stats_mode=diff[p];[s1][p]paletteuse=dither=sierra2_4a:diff_mode=rectangle:alpha_threshold=128", filterComplex)

	ffmpegArgs = append(ffmpegArgs, "-filter_complex", filterComplex)
	ffmpegArgs = append(ffmpegArgs, opts.Output)

	// Set up the command using the managed FFmpeg path
	logger.Debugf("FFmpeg command: %s %s", ffmpegPath, strings.Join(ffmpegArgs, " "))
	if rootCmd.Flag("verbose").Value.String() == "true" {
		fmt.Printf("Running FFmpeg command: %s %s\n", ffmpegPath, strings.Join(ffmpegArgs, " "))
	}
	ffmpegCmd := exec.Command(ffmpegPath, ffmpegArgs...)

	// Get pipes for stdout and stderr
	stdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := ffmpegCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Create a buffer to capture error output if needed
	var errOutput strings.Builder

	// Create a reader that both reads to our buffer and can be closed
	teeStderr := &teeReadCloser{
		Reader: io.TeeReader(stderr, &errOutput),
		Closer: stderr,
	}

	// Create a channel to receive progress updates
	progressChan := make(chan ProgressUpdate, 10)

	// Start the command
	startTime := time.Now()
	if err := ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	// Track progress in a separate goroutine
	var finalProgress *ProgressData

	if !opts.NoProgress {
		finalProgress = runProgressTracking(stdout, progressChan, startTime)
	} else {
		go trackProgress(teeStderr)
	}

	// Wait for the command to finish
	if err := ffmpegCmd.Wait(); err != nil {
		errMsg := errOutput.String()
		if len(errMsg) > 500 {
			errMsg = errMsg[len(errMsg)-500:] // Get last 500 chars
		}
		return fmt.Errorf("FFmpeg conversion failed: %w\nLast error output: %s", err, errMsg)
	}

	// Close the progress channel to signal completion
	close(progressChan)

	// Wait for the progress bar to finish updating
	time.Sleep(300 * time.Millisecond)

	elapsedTime := time.Since(startTime).Seconds()

	// Check the output file
	fileInfo, err := os.Stat(opts.Output)
	if err != nil {
		return fmt.Errorf("failed to get output file info: %w", err)
	}

	fileSizeMB := float64(fileInfo.Size()) / 1024 / 1024

	// Print summary with richer formatting
	fmt.Println()
	color.New(color.FgHiGreen, color.Bold).Println("✅ GIF created successfully!")

	// Display detailed information about the conversion
	fmt.Println()
	fmt.Println("┌─" + strings.Repeat("─", 50) + "┐")
	fmt.Printf("│ %-20s %-28s │\n", color.New(color.FgHiCyan).Sprint(" Output:"), opts.Output)
	fmt.Printf("│ %-20s %-28s │\n", color.New(color.FgHiCyan).Sprint(" Size:"), fmt.Sprintf("%.2f MB", fileSizeMB))
	fmt.Printf("│ %-20s %-28s │\n", color.New(color.FgHiCyan).Sprint(" Dimensions:"), fmt.Sprintf("%dx%d", finalProgress.Width, finalProgress.Height))
	fmt.Printf("│ %-20s %-28s │\n", color.New(color.FgHiCyan).Sprint(" Frames:"), fmt.Sprintf("%d frames at %d fps", finalProgress.Frames, opts.FPS))
	fmt.Printf("│ %-20s %-28s │\n", color.New(color.FgHiCyan).Sprint(" Conversion time:"), fmt.Sprintf("%.1f seconds", elapsedTime))
	fmt.Printf("│ %-20s %-28s │\n", color.New(color.FgHiCyan).Sprint(" Processing rate:"), fmt.Sprintf("%.2fx real-time", finalProgress.AvgProcessRate))
	fmt.Println("└─" + strings.Repeat("─", 50) + "┘")

	logger.Infof("Conversion completed: %s (%.2f MB) in %.1f seconds",
		opts.Output, fileSizeMB, elapsedTime)

	return nil
}

// Update the checkFFmpegInstallation function to use the manager
func checkFFmpegInstallation() error {
	logger := GetLogger()

	// Get FFmpeg path from the manager
	ffmpegPath, err := ffmpegManager.GetPath()
	if err != nil {
		return fmt.Errorf("FFmpeg not found. Error: %w", err)
	}

	// Test the FFmpeg binary
	cmd := exec.Command(ffmpegPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("FFmpeg not working properly. Error: %w", err)
	}

	// Log FFmpeg version
	versionStr := strings.Split(string(output), "\n")[0]
	logger.Debugf("Using FFmpeg: %s", versionStr)

	return nil
}

// ProgressUpdate represents a progress update sent through the channel
type ProgressUpdate struct {
	CurrentTime     float64
	TotalDuration   float64
	ProcessingRate  float64 // Speed relative to real-time playback
	CurrentSize     int64
	SizeUnit        string
	Bitrate         float64
	BitrateUnit     string
	FramesProcessed int64
	Width           int
	Height          int
}

// ProgressData tracks the current state of the conversion
type ProgressData struct {
	StartTime       time.Time
	CurrentTime     float64
	TotalDuration   float64
	ProcessingRate  float64 // Ratio of processing speed to real-time
	CurrentSize     int64
	SizeUnit        string
	Bitrate         float64
	BitrateUnit     string
	FramesProcessed int64
	Width           int
	Height          int
	AvgProcessRate  float64 // Average processing rate relative to real-time
	Frames          int
}

func runProgressTracking(r io.ReadCloser, progressChan chan ProgressUpdate, startTime time.Time) *ProgressData {
	// Create a new progress data struct to track conversion
	progress := &ProgressData{
		StartTime:      startTime,
		ProcessingRate: 1.0,
	}

	// Start a goroutine to parse the FFmpeg output
	go parseFFmpegOutput(r, progressChan)

	// Start a goroutine to display the progress bar
	go displayProgressBar(progressChan, progress)

	return progress
}

func parseFFmpegOutput(r io.ReadCloser, progressChan chan ProgressUpdate) {
	defer r.Close()

	// Create a scanner with a larger buffer for FFmpeg output
	scanner := bufio.NewScanner(r)
	// Increase the buffer size to handle longer lines in FFmpeg output
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Progress format patterns
	timeRegex := regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2})`)
	outTimeRegex := regexp.MustCompile(`out_time=(\d{2}:\d{2}:\d{2}\.\d{2})`)
	outTimeSecondsRegex := regexp.MustCompile(`out_time_ms=(\d+)`)
	durationRegex := regexp.MustCompile(`Duration: (\d{2}:\d{2}:\d{2}\.\d{2})`)
	totalDurationSecondsRegex := regexp.MustCompile(`duration=(\d+\.\d+)`)
	speedRegex := regexp.MustCompile(`speed=(\d+\.\d+)x`)
	sizeRegex := regexp.MustCompile(`size=\s*(\d+)(\w+)`)
	bitrateRegex := regexp.MustCompile(`bitrate=\s*(\d+\.\d+)(\w+)\/s`)
	frameRegex := regexp.MustCompile(`frame=\s*(\d+)`)
	dimensionRegex := regexp.MustCompile(`(\d+)x(\d+)`)

	// Add new regex for progress detection in numeric format
	progressRegex := regexp.MustCompile(`(\d+\.\d+)%`)

	// Current progress state
	var update ProgressUpdate
	var changed bool
	var lastLogTime time.Time

	for scanner.Scan() {
		line := scanner.Text()
		changed = false

		// Debug log to see FFmpeg output (uncomment if needed)
		// fmt.Println("FFmpeg:", line)

		// Get the duration if we don't have it yet
		if update.TotalDuration == 0 {
			if matches := durationRegex.FindStringSubmatch(line); matches != nil {
				durationStr := matches[1]
				update.TotalDuration = timeToSeconds(durationStr)
				changed = true
			} else if matches := totalDurationSecondsRegex.FindStringSubmatch(line); matches != nil {
				duration, err := strconv.ParseFloat(matches[1], 64)
				if err == nil && duration > 0 {
					update.TotalDuration = duration
					changed = true
				}
			}
		}

		// Track current time
		if matches := timeRegex.FindStringSubmatch(line); matches != nil {
			timeStr := matches[1]
			newTime := timeToSeconds(timeStr)
			if newTime > 0 {
				update.CurrentTime = newTime
				changed = true
			}
		} else if matches := outTimeRegex.FindStringSubmatch(line); matches != nil {
			timeStr := matches[1]
			newTime := timeToSeconds(timeStr)
			if newTime > 0 {
				update.CurrentTime = newTime
				changed = true
			}
		} else if matches := outTimeSecondsRegex.FindStringSubmatch(line); matches != nil {
			ms, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil && ms > 0 {
				update.CurrentTime = float64(ms) / 1000000.0
				changed = true
			}
		}

		// Check for percentage progress
		if matches := progressRegex.FindStringSubmatch(line); matches != nil {
			progress, err := strconv.ParseFloat(matches[1], 64)
			if err == nil && progress > 0 && update.TotalDuration > 0 {
				// If we have the total duration, calculate current time from percentage
				update.CurrentTime = (progress / 100.0) * update.TotalDuration
				changed = true
			}
		}

		// Track encoding speed
		if matches := speedRegex.FindStringSubmatch(line); matches != nil {
			s, err := strconv.ParseFloat(matches[1], 64)
			if err == nil && s > 0 {
				update.ProcessingRate = s
				changed = true
			}
		}

		// Track current file size
		if matches := sizeRegex.FindStringSubmatch(line); matches != nil {
			s, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil && s > 0 {
				update.CurrentSize = s
				update.SizeUnit = matches[2]
				changed = true
			}
		}

		// Track bitrate
		if matches := bitrateRegex.FindStringSubmatch(line); matches != nil {
			b, err := strconv.ParseFloat(matches[1], 64)
			if err == nil && b > 0 {
				update.Bitrate = b
				update.BitrateUnit = matches[2]
				changed = true
			}
		}

		// Track frames processed
		if matches := frameRegex.FindStringSubmatch(line); matches != nil {
			f, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil && f > 0 {
				update.FramesProcessed = f
				changed = true
			}
		}

		// Track dimensions
		if matches := dimensionRegex.FindStringSubmatch(line); matches != nil {
			w, err1 := strconv.Atoi(matches[1])
			h, err2 := strconv.Atoi(matches[2])
			if err1 == nil && err2 == nil && w > 0 && h > 0 {
				update.Width = w
				update.Height = h
				changed = true
			}
		}

		// Send update if values changed or at maximum every 200ms for any activity
		if changed || (update.FramesProcessed > 0 && time.Since(lastLogTime) > 200*time.Millisecond) {
			lastLogTime = time.Now()
			select {
			case progressChan <- update:
				// Successfully sent update
			default:
				// Channel buffer is full, skip this update
			}
		}
	}
}

func displayProgressBar(progressChan chan ProgressUpdate, progress *ProgressData) {
	// Define colors for better visual appeal
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()
	white := color.New(color.FgHiWhite).SprintFunc()

	// Terminal width for adaptive sizing
	termWidth := 80
	if width, _, err := term.GetSize(0); err == nil && width > 0 {
		termWidth = width
	}

	// Characters for the progress bar
	const (
		progressChar  = "█"
		remainingChar = "░"
		spinnerChars  = "|/-\\"
	)

	// Initialize spinner for waiting for duration
	spinnerIdx := 0
	spinner := func() string {
		s := string(spinnerChars[spinnerIdx])
		spinnerIdx = (spinnerIdx + 1) % len(spinnerChars)
		return s
	}

	// Clear the progress line
	clearLine := func() {
		fmt.Print("\r\033[K") // Clear the current line
	}

	// Initialize frame rate tracking
	frameRates := make([]float64, 0, 20)
	lastFrameCount := int64(0)
	lastFrameTime := time.Now()

	// Track frame processing rate to estimate time remaining
	firstFrameTime := time.Now()
	firstFrameCount := int64(0)
	estimatedTotalFrames := int64(0)

	// Display the initial progress bar
	fmt.Println(cyan("╭─") + strings.Repeat("─", termWidth-4) + cyan("─╮"))
	fmt.Println(cyan("│") + white(" Converting video to GIF...") + strings.Repeat(" ", termWidth-28) + cyan("│"))
	fmt.Println(cyan("│") + strings.Repeat(" ", termWidth-2) + cyan("│"))
	fmt.Println(cyan("│") + strings.Repeat(" ", termWidth-2) + cyan("│"))
	fmt.Println(cyan("│") + strings.Repeat(" ", termWidth-2) + cyan("│"))
	fmt.Println(cyan("│") + strings.Repeat(" ", termWidth-2) + cyan("│"))
	fmt.Println(cyan("╰─") + strings.Repeat("─", termWidth-4) + cyan("─╯"))

	// Move cursor up to the first data line
	fmt.Print("\033[5A")

	// Keep track of metrics for final summary
	var speedSum float64
	var speedCount int
	var lastFrame int64

	// Update progress bar every 100ms
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case update, ok := <-progressChan:
			if !ok {
				// Channel closed, conversion is complete
				return
			}

			// Update the progress struct
			progress.CurrentTime = update.CurrentTime
			progress.TotalDuration = update.TotalDuration
			progress.ProcessingRate = update.ProcessingRate
			progress.CurrentSize = update.CurrentSize
			progress.SizeUnit = update.SizeUnit
			progress.Bitrate = update.Bitrate
			progress.BitrateUnit = update.BitrateUnit

			if update.FramesProcessed > 0 {
				// Track first frame for estimation
				if firstFrameCount == 0 {
					firstFrameCount = update.FramesProcessed
					firstFrameTime = time.Now()
				}

				progress.FramesProcessed = update.FramesProcessed
				lastFrame = update.FramesProcessed
			}

			if update.Width > 0 {
				progress.Width = update.Width

				// Try to estimate total frames based on typical FPS and duration
				// A rough estimate if we have dimensions and fps information
				if estimatedTotalFrames == 0 && progress.FramesProcessed > 30 {
					// Assuming a typical video, estimate total frames
					estimatedTotalFrames = progress.FramesProcessed * 10 // Rough estimate
				}
			}

			if update.Height > 0 {
				progress.Height = update.Height
			}

			// Track for final summary
			if update.ProcessingRate > 0 {
				speedSum += update.ProcessingRate
				speedCount++
				progress.AvgProcessRate = speedSum / float64(speedCount)
			}

			// Track frame rate
			if progress.FramesProcessed > lastFrameCount {
				elapsed := time.Since(lastFrameTime).Seconds()
				if elapsed > 0 {
					fps := float64(progress.FramesProcessed-lastFrameCount) / elapsed
					frameRates = append(frameRates, fps)
					if len(frameRates) > 10 {
						frameRates = frameRates[1:]
					}
				}
				lastFrameCount = progress.FramesProcessed
				lastFrameTime = time.Now()
			}

			// Estimate final frame count for summary
			progress.Frames = int(lastFrame)

		case <-ticker.C:
			// Update the progress bar
			clearLine()

			// Calculate time elapsed
			elapsedTime := time.Since(progress.StartTime).Seconds()

			// Calculate average FPS
			avgFPS := 0.0
			if len(frameRates) > 0 {
				sum := 0.0
				for _, rate := range frameRates {
					sum += rate
				}
				avgFPS = sum / float64(len(frameRates))
			}

			// Available width for progress bar
			barWidth := termWidth - 6

			// If no duration is available yet, still show useful info
			if progress.TotalDuration <= 0 {
				// Show a spinner in the progress bar - use \r to stay on the same line
				fmt.Printf("\r\033[K %s Analyzing video...", green(spinner()))

				// Calculate estimated time remaining if we have enough information
				timeRemainingStr := "estimating..."
				if progress.FramesProcessed > 30 && avgFPS > 0 && estimatedTotalFrames > 0 {
					// Calculate a rough estimate of remaining time based on processing rate
					frameProcessingRate := float64(progress.FramesProcessed-firstFrameCount) /
						time.Since(firstFrameTime).Seconds()

					if frameProcessingRate > 0 {
						framesRemaining := float64(estimatedTotalFrames - progress.FramesProcessed)
						if framesRemaining > 0 {
							timeRemaining := framesRemaining / frameProcessingRate
							timeRemainingStr = formatDuration(timeRemaining)
						}
					}
				}

				// Only show a simplified one-line progress to prevent multiple line output
				if ticker.C != nil && progress.FramesProcessed > 0 {
					fmt.Printf("\r\033[K %s | %s elapsed | %s | %d frames | %.1f fps",
						blue("Analyzing"),
						green(formatDuration(elapsedTime)),
						yellow(timeRemainingStr),
						progress.FramesProcessed,
						avgFPS)
				}

				continue
			}

			// Calculate progress percentage
			percentage := 0.0
			if progress.TotalDuration > 0 {
				percentage = (progress.CurrentTime / progress.TotalDuration) * 100
				if percentage > 100 {
					percentage = 100
				}
			}

			// Calculate time remaining
			remainingTime := 0.0
			if progress.ProcessingRate > 0 && progress.CurrentTime > 0 {
				remainingTime = (progress.TotalDuration - progress.CurrentTime) / progress.ProcessingRate
				if remainingTime < 0 {
					remainingTime = 0
				}
			}

			// Calculate estimated final size
			estimatedSize := 0.0
			estimatedSizeLabel := ""
			if progress.CurrentSize > 0 && progress.CurrentTime > 0 && progress.TotalDuration > 0 {
				sizeMultiplier := 1.0
				switch strings.ToLower(progress.SizeUnit) {
				case "kb", "k":
					sizeMultiplier = 1.0 / 1024.0 // Convert to MB
					estimatedSizeLabel = "MB"
				case "mb", "m":
					sizeMultiplier = 1.0
					estimatedSizeLabel = "MB"
				case "gb", "g":
					sizeMultiplier = 1024.0
					estimatedSizeLabel = "MB"
				case "b":
					sizeMultiplier = 1.0 / (1024.0 * 1024.0) // Convert to MB
					estimatedSizeLabel = "MB"
				}

				currentSizeMB := float64(progress.CurrentSize) * sizeMultiplier
				if currentSizeMB > 0 && progress.CurrentTime > 0 {
					estimatedSize = currentSizeMB * (progress.TotalDuration / progress.CurrentTime)
				}
			}

			// Draw progress bar
			progressLen := int((percentage / 100) * float64(barWidth))
			if progressLen < 0 {
				progressLen = 0
			}
			if progressLen > barWidth {
				progressLen = barWidth
			}

			progressBar := green(strings.Repeat(progressChar, progressLen)) +
				yellow(strings.Repeat(remainingChar, barWidth-progressLen))

			// Line 1: Progress bar with percentage
			progressLine := fmt.Sprintf(" %s [%s] %s",
				blue("Progress:"),
				progressBar,
				yellow(fmt.Sprintf("%5.1f%%", percentage)))

			// Print the progress bar and information
			fmt.Println(progressLine)

			// Line 2: Time information
			timeLine := fmt.Sprintf(" %s %s of %s | %s remaining | %s elapsed",
				blue("Time:"),
				yellow(formatTime(progress.CurrentTime)),
				green(formatTime(progress.TotalDuration)),
				yellow(formatDuration(remainingTime)),
				green(formatDuration(elapsedTime)))

			fmt.Println(timeLine)

			// Line 3: File size and conversion information
			sizeLine := fmt.Sprintf(" %s %s | %s est. final size | %s processed",
				blue("Size:"),
				yellow(formatSize(progress.CurrentSize, progress.SizeUnit)),
				green(fmt.Sprintf("%.2f %s", estimatedSize, estimatedSizeLabel)),
				magenta(fmt.Sprintf("%d frames", progress.FramesProcessed)))

			fmt.Println(sizeLine)

			// Line 4: Technical information
			statsLine := fmt.Sprintf(" %s %s | %s | %s",
				blue("Stats:"),
				yellow(fmt.Sprintf("%.1fx processing rate", progress.ProcessingRate)),
				green(fmt.Sprintf("%.1f fps", avgFPS)),
				formatDimensions(progress.Width, progress.Height))

			fmt.Println(statsLine)

			// Move cursor back up
			fmt.Print("\033[4A")
		}
	}
}

func trackProgress(r io.ReadCloser) {
	defer r.Close()

	scanner := bufio.NewScanner(r)
	timeRegex := regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2})`)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := timeRegex.FindStringSubmatch(line); matches != nil {
			fmt.Printf("\r\033[KProgress: %s", matches[1])
		}
	}
}

// Helper function to format time in HH:MM:SS format
func formatTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// Helper function to format duration in a human-readable format
func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.0fs", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%.0fm %.0fs", seconds/60, math.Mod(seconds, 60))
	} else {
		return fmt.Sprintf("%.0fh %.0fm", seconds/3600, (math.Mod(seconds, 3600))/60)
	}
}

// Helper function to format file size
func formatSize(size int64, unit string) string {
	if size == 0 {
		return "0 KB"
	}

	// Convert to appropriate unit
	switch strings.ToLower(unit) {
	case "kb", "k":
		return fmt.Sprintf("%.2f KB", float64(size))
	case "mb", "m":
		return fmt.Sprintf("%.2f MB", float64(size))
	case "gb", "g":
		return fmt.Sprintf("%.2f GB", float64(size))
	case "b":
		if size < 1024 {
			return fmt.Sprintf("%d bytes", size)
		} else if size < 1024*1024 {
			return fmt.Sprintf("%.2f KB", float64(size)/1024)
		} else {
			return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
		}
	default:
		// If the unit is not recognized, try to determine appropriate unit
		if size < 1024 {
			return fmt.Sprintf("%d bytes", size)
		} else if size < 1024*1024 {
			return fmt.Sprintf("%.2f KB", float64(size)/1024)
		} else if size < 1024*1024*1024 {
			return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
		} else {
			return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
		}
	}
}

// Convert time string in format HH:MM:SS.MS to seconds
func timeToSeconds(timeStr string) float64 {
	var h, m, s, ms float64
	parts := strings.Split(timeStr, ":")
	if len(parts) == 3 {
		fmt.Sscanf(parts[0], "%f", &h)
		fmt.Sscanf(parts[1], "%f", &m)
		secParts := strings.Split(parts[2], ".")
		fmt.Sscanf(secParts[0], "%f", &s)
		if len(secParts) > 1 {
			msStr := secParts[1]
			fmt.Sscanf(msStr, "%f", &ms)
			ms = ms / math.Pow10(len(msStr))
		}
	}
	return h*3600 + m*60 + s + ms
}

// teeReadCloser combines a Reader and Closer to implement ReadCloser
type teeReadCloser struct {
	io.Reader
	io.Closer
}

// Helper function to format dimensions
func formatDimensions(width, height int) string {
	magenta := color.New(color.FgMagenta).SprintFunc()
	if width > 0 && height > 0 {
		return magenta(fmt.Sprintf("%dx%d", width, height))
	}
	return magenta("analyzing...")
}
