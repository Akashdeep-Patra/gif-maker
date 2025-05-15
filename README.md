# GIF-Maker

A production-grade CLI tool for converting video files to GIFs with customizable options and real-time progress tracking.

## Features

- **Flexible Video Conversion**: Convert video files to optimized GIFs with customizable parameters
- **Interactive Mode**: Guided process with file picker support and sensible defaults
- **Customization Options**: Control quality, size, frame rate, start time, and duration
- **Real-time Progress Tracking**: Visual progress bar with time estimation and statistics
- **Video Analysis**: Display detailed information about video files
- **Cross-platform Support**: Works on macOS, Windows, and Linux
- **Embedded FFmpeg**: Uses system FFmpeg or falls back to embedded binaries

## Prerequisites

This tool requires FFmpeg to be installed on your system for optimal performance:

- **macOS**: `brew install ffmpeg`
- **Ubuntu/Debian**: `sudo apt install ffmpeg`
- **Windows**: Download from [ffmpeg.org](https://ffmpeg.org/download.html)

If FFmpeg is not installed, the tool will attempt to use embedded binaries (if available).

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/Akashdeep-Patra/gif-maker.git
cd gif-maker

# Build the binary
go build -o gif-maker

# Optional: Move to a directory in your PATH
mv gif-maker /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/Akashdeep-Patra/gif-maker@latest
```

## Usage

### Basic Commands

```bash
# Show help information
gif-maker --help

# Check version information
gif-maker version

# Get information about a video file
gif-maker info path/to/video.mp4

# Convert a video to GIF (with default settings)
gif-maker convert -i path/to/video.mp4 -o output.gif

# Use interactive mode
gif-maker convert --interactive

# Convert with specific options
gif-maker convert -i input.mp4 -o output.gif --fps 15 --width 500 --quality 80 --start 00:00:10 --duration 00:00:05
```

## Command Reference

### Convert Command

```
gif-maker convert [flags]
```

The convert command transforms video files into optimized GIFs with customizable parameters.

#### Flags

- `-i, --input string`: Input video file path (required unless using interactive mode)
- `-o, --output string`: Output GIF file path (default: input_name.gif)
- `-f, --fps int`: Frames per second (default 10) - higher values create smoother animations but larger files
- `--start string`: Start time in format HH:MM:SS (e.g., 00:01:30 for 1 minute 30 seconds)
- `--duration string`: Duration in format HH:MM:SS (how much of the video to convert)
- `-w, --width int`: Output width in pixels (height is calculated automatically to maintain aspect ratio)
- `-q, --quality int`: Output quality from 1-100 (default 90) - higher values produce better colors but larger files
- `-I, --interactive`: Use interactive mode with guided prompts (default if no arguments provided)
- `--no-progress`: Disable the progress bar (useful for scripts or CI/CD pipelines)
- `-v, --verbose`: Enable verbose logging (writes detailed logs to a temporary file)

#### Interactive Mode

When run with the `--interactive` flag or without any arguments, the convert command enters interactive mode:

1. **File Selection**: Offers to use a graphical file picker or manual path entry
2. **Output Configuration**: Prompts for output file location and name
3. **Quality Settings**: Options for FPS, dimensions, and quality presets
4. **Time Selection**: Options to specify start time and duration

#### Conversion Process

The conversion process involves several stages:

1. **Initialization**: Validates input file and sets up default output if needed
2. **FFmpeg Configuration**: Builds an optimized FFmpeg command with appropriate filters
3. **Palette Generation**: Creates a custom color palette for better GIF quality
4. **Conversion**: Processes the video through FFmpeg with real-time progress tracking
5. **Optimization**: Applies dithering and compression techniques for optimal file size
6. **Completion**: Displays summary statistics about the resulting GIF

### Info Command

```
gif-maker info [video file]
```

The info command analyzes video files and displays detailed information.

#### Output Information

- **File Size**: Total size of the video file
- **Resolution**: Width and height in pixels
- **Duration**: Total length in minutes, seconds, and milliseconds
- **Frame Rate**: Frames per second (FPS)
- **Estimated GIF Sizes**: Approximations of resulting GIF sizes at different FPS settings

#### Technical Process

The info command:
1. Uses FFmpeg's ffprobe to extract video metadata
2. Parses the output to retrieve width, height, duration, and frame rate
3. Calculates estimated GIF sizes based on pixel count, duration, and different FPS values
4. Formats the information in a user-friendly display

### Version Command

```
gif-maker version
```

Displays version information for the application and verifies FFmpeg installation.

## Technical Details

### FFmpeg Integration

The application integrates with FFmpeg in several ways:

1. **FFmpeg Detection**: The tool first attempts to find FFmpeg in the system PATH
2. **Embedded Binaries**: If system FFmpeg is not available, it extracts and uses embedded binaries
3. **Command Construction**: Builds optimized FFmpeg commands for video processing with:
   - Palette generation for better color accuracy
   - Filter chains for resizing and frame rate adjustment
   - Dithering algorithms for better visual quality
   - Multi-threading for improved performance

### Progress Tracking System

The progress tracking system provides real-time feedback during conversion:

1. **FFmpeg Output Parsing**: Analyzes FFmpeg's output to extract:
   - Current time position
   - Processing speed relative to real-time
   - File size and bitrate
   - Frame count and dimensions

2. **Visual Progress Bar**: Displays:
   - Percentage completion
   - Elapsed and remaining time estimates
   - Current and estimated final file size
   - Processing statistics (speed, frame rate)
   - Video dimensions

3. **Adaptive UI**: The progress bar adapts to terminal size and capabilities

### File Picker Integration

On supported platforms, the tool provides a native file picker:

- **macOS**: Uses AppleScript to display native file dialogs
- **Windows**: Uses PowerShell and Windows Forms to create dialogs
- **Linux**: Uses zenity if available for GTK-based dialogs

The file picker provides:
- Input file selection with video file filtering
- Output location selection with default GIF extension
- Fallback to text input if graphical selection fails

## Project Architecture

### Directory Structure

```
├── cmd/                  # Command implementations
│   ├── convert.go        # Video to GIF conversion functionality
│   ├── info.go           # Video information display
│   ├── root.go           # Root command and shared functionality 
│   ├── util.go           # Utility functions
│   └── version.go        # Version information
├── internal/             # Internal packages
│   └── ffmpeg/           # FFmpeg management
│       ├── ffmpeg.go     # FFmpeg binary handling
│       └── binaries/     # Embedded FFmpeg binaries
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
└── main.go               # Application entry point
```

### Key Components

#### FFmpeg Manager (`internal/ffmpeg/ffmpeg.go`)

The FFmpeg Manager handles:
1. **Binary Detection**: Locates system-installed FFmpeg
2. **Binary Extraction**: Extracts embedded binaries if needed
3. **Path Management**: Provides the path to the appropriate FFmpeg binary
4. **Cleanup**: Removes temporary files when done

#### Convert Command (`cmd/convert.go`)

The convert command implements:
1. **Option Parsing**: Handles command-line flags and defaults
2. **Interactive Mode**: Implements user-friendly prompts
3. **File Picker**: Provides native file selection dialogs
4. **FFmpeg Command Construction**: Builds optimized FFmpeg commands
5. **Progress Tracking**: Implements real-time progress display
6. **GIF Optimization**: Applies techniques for better quality at smaller sizes

#### Utility Functions (`cmd/util.go`)

Provides shared functionality:
1. **FFmpeg Detection**: Verifies FFmpeg availability
2. **Video Analysis**: Extracts information from video files
3. **Resource Optimization**: Determines optimal thread count
4. **Format Helpers**: Converts bytes to human-readable formats
5. **Validation**: Verifies time format strings

## Advanced Usage Examples

### Creating a High-Quality GIF

```bash
gif-maker convert -i video.mp4 -o high_quality.gif --fps 20 --width 800 --quality 95
```

This creates a high-quality GIF with:
- 20 frames per second for smooth animation
- 800 pixels wide (height calculated to maintain aspect ratio)
- 95% quality setting for better color reproduction

### Extracting a Specific Video Segment

```bash
gif-maker convert -i video.mp4 -o clip.gif --start 00:01:30 --duration 00:00:10 --fps 15
```

This extracts a 10-second clip starting at 1 minute and 30 seconds into the video, converted at 15 fps.

### Creating a Small, Optimized GIF

```bash
gif-maker convert -i video.mp4 -o small.gif --fps 8 --width 320 --quality 60
```

This creates a smaller, more optimized GIF with:
- Reduced frame rate of 8 fps
- Smaller dimensions (320 pixels wide)
- Lower quality setting for reduced file size

## Troubleshooting

### Common Issues

#### "FFmpeg not found in PATH"

**Solution**: Install FFmpeg using the instructions in the Prerequisites section, or ensure it's in your system PATH.

#### "Failed to convert video"

Possible causes and solutions:
1. **Invalid input file**: Verify the video file exists and is a supported format
2. **Permission issues**: Ensure you have read access to the input file and write access to the output directory
3. **Invalid time format**: Ensure start time and duration use the HH:MM:SS format

#### Progress bar displays incorrectly

**Solution**: Use the `--no-progress` flag to disable the progress bar.

### Logging

The application creates logs in a temporary directory:
- macOS/Linux: `/tmp/gif-maker-logs/gif-maker.log`
- Windows: `%TEMP%\gif-maker-logs\gif-maker.log`

Use the `--verbose` flag to enable detailed logging for troubleshooting.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details. 