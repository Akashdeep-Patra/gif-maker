# FFmpeg Binaries

This directory is used to store FFmpeg binaries for different platforms.

## Adding FFmpeg Binaries

1. Download the appropriate FFmpeg binaries for each platform:
   - Windows: https://www.gyan.dev/ffmpeg/builds/ or https://ffmpeg.org/download.html#build-windows
   - macOS: https://evermeet.cx/ffmpeg/ or https://ffmpeg.org/download.html#build-mac
   - Linux: https://johnvansickle.com/ffmpeg/ or use the appropriate package manager

2. Rename the binaries according to the following convention:
   - Windows x64: `ffmpeg-win64.exe`
   - Windows x86: `ffmpeg-win32.exe`
   - macOS Intel: `ffmpeg-macos-x86_64`
   - macOS Apple Silicon: `ffmpeg-macos-arm64`
   - Linux x64: `ffmpeg-linux-x86_64`
   - Linux x86: `ffmpeg-linux-i386`
   - Linux ARM64: `ffmpeg-linux-arm64`
   - Linux ARM: `ffmpeg-linux-armhf`

3. Place the renamed binaries in this directory.

## License

Make sure to comply with FFmpeg's license requirements when distributing these binaries.
See the [FFmpeg license](https://ffmpeg.org/legal.html) for more information. 