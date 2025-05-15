#!/bin/bash
#
# Script to download FFmpeg binaries for packaging with the application
# Only downloads the binary for the current platform
#

set -e

# Target directory
TARGET_DIR="internal/ffmpeg/binaries"
mkdir -p "$TARGET_DIR"

# Create a temp directory for downloads
TEMP_DIR=$(mktemp -d)
echo "Using temporary directory: $TEMP_DIR"

function cleanup() {
  echo "Cleaning up..."
  rm -rf "$TEMP_DIR"
  echo "Done!"
}

trap cleanup EXIT

# Detect current platform
PLATFORM="unknown"
ARCH="unknown"

# Detect OS
if [[ "$OSTYPE" == "darwin"* ]]; then
  PLATFORM="macos"
  # Detect macOS architecture
  if [[ "$(uname -m)" == "arm64" ]]; then
    ARCH="arm64"
  else
    ARCH="x86_64"
  fi
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
  PLATFORM="linux"
  # Detect Linux architecture
  if [[ "$(uname -m)" == "x86_64" ]]; then
    ARCH="x86_64"
  elif [[ "$(uname -m)" == "i686" ]]; then
    ARCH="i386"
  elif [[ "$(uname -m)" == "aarch64" ]]; then
    ARCH="arm64"
  elif [[ "$(uname -m)" == "armv7"* ]]; then
    ARCH="armhf"
  fi
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" || "$OSTYPE" == "cygwin" ]]; then
  PLATFORM="windows"
  # Detect Windows architecture
  if [[ "$(uname -m)" == "x86_64" ]]; then
    ARCH="win64"
  else
    ARCH="win32"
  fi
fi

# Check if platform was identified
if [[ "$PLATFORM" == "unknown" || "$ARCH" == "unknown" ]]; then
  echo "Error: Could not detect your platform or architecture."
  echo "Please manually download the appropriate FFmpeg binary for your system."
  echo "For more information, see: internal/ffmpeg/binaries/README.md"
  exit 1
fi

echo "Detected platform: $PLATFORM"
echo "Detected architecture: $ARCH"
echo "Will download FFmpeg for: $PLATFORM-$ARCH"

# Download the appropriate binary
case "$PLATFORM" in
  "macos")
    echo "Downloading macOS binary ($ARCH)..."
    if [[ "$ARCH" == "arm64" ]]; then
      # For Apple Silicon, just download the universal binary instead
      # The specific ARM URL is not reliable
      echo "Downloading macOS universal binary (works on ARM64)..."
      curl -L "https://evermeet.cx/ffmpeg/ffmpeg-6.1.zip" -o "$TEMP_DIR/ffmpeg-macos.zip"
      
      if [ $? -eq 0 ] && [ -s "$TEMP_DIR/ffmpeg-macos.zip" ]; then
        unzip -q "$TEMP_DIR/ffmpeg-macos.zip" -d "$TEMP_DIR" || echo "Warning: unzip failed, may not be a valid zip file"
        if [ -f "$TEMP_DIR/ffmpeg" ]; then
          mv "$TEMP_DIR/ffmpeg" "$TARGET_DIR/ffmpeg-macos-arm64"
          chmod +x "$TARGET_DIR/ffmpeg-macos-arm64"
          echo "macOS ARM64 binary downloaded and installed."
        else
          echo "Error: Extracted file not found"
          exit 1
        fi
      else
        echo "Failed to download macOS binary, using alternative source..."
        # Fallback to the static builds site
        curl -L "https://www.osxexperts.net/ffmpeg/ffmpeg-arm64.zip" -o "$TEMP_DIR/ffmpeg-macos-alt.zip"
        if [ $? -eq 0 ] && [ -s "$TEMP_DIR/ffmpeg-macos-alt.zip" ]; then
          unzip -q "$TEMP_DIR/ffmpeg-macos-alt.zip" -d "$TEMP_DIR" || echo "Warning: unzip failed"
          if [ -f "$TEMP_DIR/ffmpeg" ]; then
            mv "$TEMP_DIR/ffmpeg" "$TARGET_DIR/ffmpeg-macos-arm64"
            chmod +x "$TARGET_DIR/ffmpeg-macos-arm64"
            echo "macOS ARM64 binary downloaded and installed from alternative source."
          else
            echo "Error: Could not extract ffmpeg from alternative source"
            exit 1
          fi
        else
          echo "Error: All download attempts failed. Please download FFmpeg manually."
          exit 1
        fi
      fi
    else
      echo "Downloading macOS Intel binary..."
      curl -L "https://evermeet.cx/ffmpeg/ffmpeg-6.1.zip" -o "$TEMP_DIR/ffmpeg-macos-x86_64.zip"
      
      if [ $? -eq 0 ] && [ -s "$TEMP_DIR/ffmpeg-macos-x86_64.zip" ]; then
        unzip -q "$TEMP_DIR/ffmpeg-macos-x86_64.zip" -d "$TEMP_DIR" || echo "Warning: unzip failed"
        if [ -f "$TEMP_DIR/ffmpeg" ]; then
          mv "$TEMP_DIR/ffmpeg" "$TARGET_DIR/ffmpeg-macos-x86_64"
          chmod +x "$TARGET_DIR/ffmpeg-macos-x86_64"
          echo "macOS Intel binary downloaded and installed."
        else
          echo "Error: Extracted file not found"
          exit 1
        fi
      else
        echo "Error: Failed to download macOS Intel binary."
        exit 1
      fi
    fi
    ;;
    
  "linux")
    if [[ "$ARCH" == "x86_64" ]]; then
      echo "Downloading Linux x86_64 binary..."
      curl -L https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz -o "$TEMP_DIR/ffmpeg-linux-x86_64.tar.xz"
      
      if [ $? -eq 0 ] && [ -s "$TEMP_DIR/ffmpeg-linux-x86_64.tar.xz" ]; then
        mkdir -p "$TEMP_DIR/linux-x86_64"
        tar -xf "$TEMP_DIR/ffmpeg-linux-x86_64.tar.xz" -C "$TEMP_DIR/linux-x86_64" --strip-components 1
        cp "$TEMP_DIR/linux-x86_64/ffmpeg" "$TARGET_DIR/ffmpeg-linux-x86_64"
        chmod +x "$TARGET_DIR/ffmpeg-linux-x86_64"
        echo "Linux x86_64 binary downloaded and installed."
      else
        echo "Error: Failed to download Linux x86_64 binary."
        exit 1
      fi
    elif [[ "$ARCH" == "i386" ]]; then
      echo "Downloading Linux i386 binary..."
      curl -L https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-i686-static.tar.xz -o "$TEMP_DIR/ffmpeg-linux-i386.tar.xz"
      
      if [ $? -eq 0 ] && [ -s "$TEMP_DIR/ffmpeg-linux-i386.tar.xz" ]; then
        mkdir -p "$TEMP_DIR/linux-i386"
        tar -xf "$TEMP_DIR/ffmpeg-linux-i386.tar.xz" -C "$TEMP_DIR/linux-i386" --strip-components 1
        cp "$TEMP_DIR/linux-i386/ffmpeg" "$TARGET_DIR/ffmpeg-linux-i386"
        chmod +x "$TARGET_DIR/ffmpeg-linux-i386"
        echo "Linux i386 binary downloaded and installed."
      else
        echo "Error: Failed to download Linux i386 binary."
        exit 1
      fi
    elif [[ "$ARCH" == "arm64" ]]; then
      echo "Downloading Linux ARM64 binary..."
      curl -L https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-arm64-static.tar.xz -o "$TEMP_DIR/ffmpeg-linux-arm64.tar.xz"
      
      if [ $? -eq 0 ] && [ -s "$TEMP_DIR/ffmpeg-linux-arm64.tar.xz" ]; then
        mkdir -p "$TEMP_DIR/linux-arm64"
        tar -xf "$TEMP_DIR/ffmpeg-linux-arm64.tar.xz" -C "$TEMP_DIR/linux-arm64" --strip-components 1
        cp "$TEMP_DIR/linux-arm64/ffmpeg" "$TARGET_DIR/ffmpeg-linux-arm64"
        chmod +x "$TARGET_DIR/ffmpeg-linux-arm64"
        echo "Linux ARM64 binary downloaded and installed."
      else
        echo "Error: Failed to download Linux ARM64 binary."
        exit 1
      fi
    elif [[ "$ARCH" == "armhf" ]]; then
      echo "Downloading Linux ARMhf binary..."
      curl -L https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-armhf-static.tar.xz -o "$TEMP_DIR/ffmpeg-linux-armhf.tar.xz"
      
      if [ $? -eq 0 ] && [ -s "$TEMP_DIR/ffmpeg-linux-armhf.tar.xz" ]; then
        mkdir -p "$TEMP_DIR/linux-armhf"
        tar -xf "$TEMP_DIR/ffmpeg-linux-armhf.tar.xz" -C "$TEMP_DIR/linux-armhf" --strip-components 1
        cp "$TEMP_DIR/linux-armhf/ffmpeg" "$TARGET_DIR/ffmpeg-linux-armhf"
        chmod +x "$TARGET_DIR/ffmpeg-linux-armhf"
        echo "Linux ARMhf binary downloaded and installed."
      else
        echo "Error: Failed to download Linux ARMhf binary."
        exit 1
      fi
    fi
    ;;
    
  "windows")
    echo "Windows binary download not automated."
    echo "Please manually download and install Windows FFmpeg binary:"
    echo "1. Download from https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
    echo "2. Extract and rename the ffmpeg.exe to ffmpeg-$ARCH.exe"
    echo "3. Copy it to $TARGET_DIR"
    
    # Create a placeholder file to indicate manual action needed
    echo "# Manual action required: Please download Windows FFmpeg binary" > "$TARGET_DIR/DOWNLOAD_WINDOWS_BINARY.txt"
    ;;
esac

echo "Binary for $PLATFORM-$ARCH has been downloaded and placed in $TARGET_DIR."
echo "If building for multiple platforms, you'll need to run this script on each target platform or download binaries manually."

# Create a version file to track which binary was downloaded
echo "platform=$PLATFORM" > "$TARGET_DIR/binary_info.txt"
echo "arch=$ARCH" >> "$TARGET_DIR/binary_info.txt"
echo "timestamp=$(date)" >> "$TARGET_DIR/binary_info.txt" 