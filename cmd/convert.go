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
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"

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

	// Create progress data
	progress := &ProgressData{
		StartTime:      time.Now(),
		ProcessingRate: 1.0,
	}

	// Find the total duration from the input file
	totalDuration, videoDimensions, err := getVideoMetadata(opts.Input, ffmpegPath)
	if err != nil {
		logger.Warnf("Could not get video metadata: %v", err)
	}

	if videoDimensions[0] > 0 && videoDimensions[1] > 0 {
		progress.Width = videoDimensions[0]
		progress.Height = videoDimensions[1]
	}

	if totalDuration > 0 {
		progress.TotalDuration = totalDuration
	}

	// Start the command
	startTime := time.Now()
	if err := ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	if !opts.NoProgress {
		// Create and start the progress tracking
		runMPBProgressTracking(stdout, progress, totalDuration)
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
	fmt.Printf("│ %-20s %-28s │\n", color.New(color.FgHiCyan).Sprint(" Dimensions:"), fmt.Sprintf("%dx%d", progress.Width, progress.Height))
	fmt.Printf("│ %-20s %-28s │\n", color.New(color.FgHiCyan).Sprint(" Frames:"), fmt.Sprintf("%d frames at %d fps", progress.Frames, opts.FPS))
	fmt.Printf("│ %-20s %-28s │\n", color.New(color.FgHiCyan).Sprint(" Conversion time:"), fmt.Sprintf("%.1f seconds", elapsedTime))
	fmt.Printf("│ %-20s %-28s │\n", color.New(color.FgHiCyan).Sprint(" Processing rate:"), fmt.Sprintf("%.2fx real-time", progress.AvgProcessRate))
	fmt.Println("└─" + strings.Repeat("─", 50) + "┘")

	logger.Infof("Conversion completed: %s (%.2f MB) in %.1f seconds",
		opts.Output, fileSizeMB, elapsedTime)

	return nil
}

// Get video metadata (duration and dimensions) using FFmpeg
func getVideoMetadata(videoPath, ffmpegPath string) (float64, [2]int, error) {
	// Run ffmpeg -i input.mp4 command to get metadata
	cmd := exec.Command(ffmpegPath, "-i", videoPath)
	var out strings.Builder
	cmd.Stderr = &out
	cmd.Run() // We expect this to "fail" but give us info in stderr

	output := out.String()

	// Extract duration
	durationRegex := regexp.MustCompile(`Duration: (\d{2}):(\d{2}):(\d{2})\.(\d{2})`)
	durationMatches := durationRegex.FindStringSubmatch(output)

	var duration float64
	if len(durationMatches) >= 5 {
		hours, _ := strconv.Atoi(durationMatches[1])
		minutes, _ := strconv.Atoi(durationMatches[2])
		seconds, _ := strconv.Atoi(durationMatches[3])
		milliseconds, _ := strconv.Atoi(durationMatches[4])

		duration = float64(hours)*3600 + float64(minutes)*60 + float64(seconds) + float64(milliseconds)/100.0
	}

	// Extract video dimensions
	dimensionRegex := regexp.MustCompile(`Stream #.*Video:.* (\d+)x(\d+)`)
	dimensionMatches := dimensionRegex.FindStringSubmatch(output)

	var dimensions [2]int
	if len(dimensionMatches) >= 3 {
		dimensions[0], _ = strconv.Atoi(dimensionMatches[1])
		dimensions[1], _ = strconv.Atoi(dimensionMatches[2])
	}

	return duration, dimensions, nil
}

// New progress tracking function using MPB
func runMPBProgressTracking(r io.ReadCloser, progress *ProgressData, totalDuration float64) {
	// Create a new MPB progress container
	p := mpb.New(
		mpb.WithWidth(80),
		mpb.WithRefreshRate(100*time.Millisecond),
	)

	// Create a total bar for overall progress
	total := int64(totalDuration * 100) // Convert to centiseconds for smoother progress
	if total <= 0 {
		total = 100 // Default if we can't determine the duration
	}

	// Progress bar for encoding
	bar := p.AddBar(total,
		mpb.PrependDecorators(
			decor.Name("Converting: ", decor.WC{W: 12, C: decor.DidentRight}),
			decor.CountersNoUnit("%d / %d", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
			decor.Name(" • ", decor.WCSyncWidthR),
			decor.Elapsed(decor.ET_STYLE_GO, decor.WCSyncWidth),
			decor.Name(" • ", decor.WCSyncWidthR),
			decor.AverageSpeed(0, "%.1fx", decor.WCSyncWidth),
		),
	)

	// Status bar for file info
	statusBar := p.AddBar(0,
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(
			decor.Any(func(statistics decor.Statistics) string {
				if progress.Width > 0 && progress.Height > 0 {
					return fmt.Sprintf("Size: %s • %dx%d",
						formatSize(progress.CurrentSize, progress.SizeUnit),
						progress.Width,
						progress.Height)
				}
				return fmt.Sprintf("Size: %s", formatSize(progress.CurrentSize, progress.SizeUnit))
			}, decor.WCSyncSpaceR),
		),
	)

	// Process frame info
	frameBar := p.AddBar(0,
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(
			decor.Any(func(statistics decor.Statistics) string {
				return fmt.Sprintf("Frames: %d processed", progress.FramesProcessed)
			}, decor.WCSyncSpaceR),
		),
	)

	// Start a goroutine to parse FFmpeg output
	go func() {
		defer r.Close()

		// Track average processing rate
		var speedSum float64
		var speedCount int

		// Create a scanner with a larger buffer for FFmpeg output
		scanner := bufio.NewScanner(r)
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
		frameRegex := regexp.MustCompile(`frame=\s*(\d+)`)
		dimensionRegex := regexp.MustCompile(`(\d+)x(\d+)`)

		for scanner.Scan() {
			line := scanner.Text()

			// Track current time
			if matches := timeRegex.FindStringSubmatch(line); matches != nil {
				timeStr := matches[1]
				newTime := timeToSeconds(timeStr)
				if newTime > 0 {
					progress.CurrentTime = newTime
					// Update the progress bar
					if totalDuration > 0 {
						bar.SetCurrent(int64(newTime * 100))
					} else {
						// If we don't know the total duration, just increment
						bar.IncrInt64(1)
					}
				}
			} else if matches := outTimeRegex.FindStringSubmatch(line); matches != nil {
				timeStr := matches[1]
				newTime := timeToSeconds(timeStr)
				if newTime > 0 {
					progress.CurrentTime = newTime
					if totalDuration > 0 {
						bar.SetCurrent(int64(newTime * 100))
					} else {
						bar.IncrInt64(1)
					}
				}
			} else if matches := outTimeSecondsRegex.FindStringSubmatch(line); matches != nil {
				ms, err := strconv.ParseInt(matches[1], 10, 64)
				if err == nil && ms > 0 {
					progress.CurrentTime = float64(ms) / 1000000.0
					if totalDuration > 0 {
						bar.SetCurrent(int64(progress.CurrentTime * 100))
					} else {
						bar.IncrInt64(1)
					}
				}
			}

			// Get the duration if we don't have it yet
			if progress.TotalDuration == 0 {
				if matches := durationRegex.FindStringSubmatch(line); matches != nil {
					durationStr := matches[1]
					progress.TotalDuration = timeToSeconds(durationStr)
					if progress.TotalDuration > 0 {
						bar.SetTotal(int64(progress.TotalDuration*100), false)
					}
				} else if matches := totalDurationSecondsRegex.FindStringSubmatch(line); matches != nil {
					duration, err := strconv.ParseFloat(matches[1], 64)
					if err == nil && duration > 0 {
						progress.TotalDuration = duration
						if progress.TotalDuration > 0 {
							bar.SetTotal(int64(progress.TotalDuration*100), false)
						}
					}
				}
			}

			// Track encoding speed
			if matches := speedRegex.FindStringSubmatch(line); matches != nil {
				s, err := strconv.ParseFloat(matches[1], 64)
				if err == nil && s > 0 {
					progress.ProcessingRate = s

					// Track for final summary
					speedSum += s
					speedCount++
					progress.AvgProcessRate = speedSum / float64(speedCount)
				}
			}

			// Track current file size
			if matches := sizeRegex.FindStringSubmatch(line); matches != nil {
				s, err := strconv.ParseInt(matches[1], 10, 64)
				if err == nil && s > 0 {
					progress.CurrentSize = s
					progress.SizeUnit = matches[2]
					statusBar.SetTotal(int64(s+1), false)
					statusBar.SetCurrent(int64(s))
				}
			}

			// Track frames processed
			if matches := frameRegex.FindStringSubmatch(line); matches != nil {
				f, err := strconv.ParseInt(matches[1], 10, 64)
				if err == nil && f > 0 {
					progress.FramesProcessed = f
					progress.Frames = int(f)
					frameBar.SetTotal(int64(f+1), false)
					frameBar.SetCurrent(int64(f))
				}
			}

			// Track dimensions
			if matches := dimensionRegex.FindStringSubmatch(line); matches != nil {
				w, err1 := strconv.Atoi(matches[1])
				h, err2 := strconv.Atoi(matches[2])
				if err1 == nil && err2 == nil && w > 0 && h > 0 {
					progress.Width = w
					progress.Height = h
				}
			}
		}

		// Make sure the bars are completed when done
		if bar.Current() < 100 {
			bar.SetTotal(bar.Current(), true)
		}
		statusBar.SetTotal(statusBar.Current(), true)
		frameBar.SetTotal(frameBar.Current(), true)
	}()
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
