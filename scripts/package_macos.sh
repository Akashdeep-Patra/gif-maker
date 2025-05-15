#!/bin/bash
#
# Script to package the application for macOS distribution
#

set -e

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
APP_NAME="GIF Maker"
IDENTIFIER="com.example.gifmaker"
BUILD_DIR="./build"
DIST_DIR="./dist"
APP_BUNDLE_PATH="$DIST_DIR/$APP_NAME.app"
DMG_PATH="$DIST_DIR/GIF-Maker-$VERSION.dmg"

# Create directories
mkdir -p "$DIST_DIR"
mkdir -p "$APP_BUNDLE_PATH/Contents/MacOS"
mkdir -p "$APP_BUNDLE_PATH/Contents/Resources"

# Check if the binary exists
if [ ! -f "$BUILD_DIR/gif-maker" ]; then
  echo "Building application..."
  make build
fi

# Copy binary
cp "$BUILD_DIR/gif-maker" "$APP_BUNDLE_PATH/Contents/MacOS/"

# Create Info.plist
cat > "$APP_BUNDLE_PATH/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>English</string>
	<key>CFBundleExecutable</key>
	<string>gif-maker</string>
	<key>CFBundleIconFile</key>
	<string>AppIcon</string>
	<key>CFBundleIdentifier</key>
	<string>${IDENTIFIER}</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>${APP_NAME}</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>${VERSION}</string>
	<key>CFBundleVersion</key>
	<string>${VERSION}</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSHumanReadableCopyright</key>
	<string>Copyright © $(date +%Y), Your Name Here. All rights reserved.</string>
</dict>
</plist>
EOF

# Create a simple icon if none exists
if [ ! -f "assets/AppIcon.icns" ]; then
  echo "No app icon found. Using a default placeholder."
  mkdir -p "assets"
  # This just uses a placeholder icon message - in reality you would need to create an .icns file
  echo "You should replace this with a real icon file." > "$APP_BUNDLE_PATH/Contents/Resources/AppIcon.txt"
else
  cp "assets/AppIcon.icns" "$APP_BUNDLE_PATH/Contents/Resources/"
fi

# Create a simple launcher script
cat > "$APP_BUNDLE_PATH/Contents/MacOS/launcher" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
./gif-maker "$@"
EOF
chmod +x "$APP_BUNDLE_PATH/Contents/MacOS/launcher"

# Sign the application (optional, but recommended)
if command -v codesign &> /dev/null; then
  echo "Would you like to code sign the application? (y/n)"
  read -r sign_response
  if [[ "$sign_response" =~ ^[Yy]$ ]]; then
    echo "Available signing identities:"
    security find-identity -v -p codesigning
    echo "Enter the identity to use (leave blank to skip):"
    read -r sign_identity
    if [ -n "$sign_identity" ]; then
      echo "Signing application..."
      codesign --force --deep --sign "$sign_identity" "$APP_BUNDLE_PATH"
      echo "Application signed successfully."
    fi
  fi
fi

# Create a DMG
echo "Creating DMG..."
if command -v hdiutil &> /dev/null; then
  # Create a temporary directory for DMG contents
  TMP_DMG_DIR=$(mktemp -d)
  cp -R "$APP_BUNDLE_PATH" "$TMP_DMG_DIR"
  
  # Create a link to the Applications folder
  ln -s /Applications "$TMP_DMG_DIR/Applications"
  
  # Create the DMG
  hdiutil create -volname "GIF Maker" -srcfolder "$TMP_DMG_DIR" -ov -format UDZO "$DMG_PATH"
  
  # Clean up
  rm -rf "$TMP_DMG_DIR"
  
  echo "DMG created at: $DMG_PATH"
else
  echo "hdiutil not found. Cannot create DMG."
fi

echo "Application bundle created at: $APP_BUNDLE_PATH"
echo ""
echo "To distribute this application:"
echo "1. Send the DMG file to other Mac users: $DMG_PATH"
echo "2. They can open it and drag the application to their Applications folder"
echo ""
echo "Note: For proper distribution, you should:"
echo "- Create a proper app icon (icns file)"
echo "- Code sign the application with a Developer ID certificate from Apple"
echo "- Consider notarizing the app with Apple for maximum compatibility" 