#!/bin/bash
#
# Script to create a Homebrew formula for GIF Maker
#

set -e

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
REPO_URL="https://github.com/akashdeep/gif-maker"  # Replace with your actual GitHub repo
DIST_DIR="./dist"
FORMULA_PATH="$DIST_DIR/gif-maker.rb"

# Ensure the dist directory exists
mkdir -p "$DIST_DIR"

# Build the app if it doesn't exist
if [ ! -f "./build/gif-maker" ]; then
  echo "Building application..."
  make build
fi

# Create a tarball of the repository for the formula
echo "Creating tarball for Homebrew formula..."
git archive --format=tar.gz -o "$DIST_DIR/gif-maker-$VERSION.tar.gz" HEAD || {
  echo "Failed to create git archive. Using simple tarball instead."
  mkdir -p "$DIST_DIR/gif-maker-$VERSION"
  cp -R ./* "$DIST_DIR/gif-maker-$VERSION/"
  tar -czf "$DIST_DIR/gif-maker-$VERSION.tar.gz" -C "$DIST_DIR" "gif-maker-$VERSION"
  rm -rf "$DIST_DIR/gif-maker-$VERSION"
}

# Calculate SHA256 checksum
SHA=$(shasum -a 256 "$DIST_DIR/gif-maker-$VERSION.tar.gz" | cut -d ' ' -f 1)

# Create the Homebrew formula
cat > "$FORMULA_PATH" << EOF
class GifMaker < Formula
  desc "Command-line tool to convert videos to GIFs"
  homepage "$REPO_URL"
  url "$REPO_URL/releases/download/v$VERSION/gif-maker-$VERSION.tar.gz"
  sha256 "$SHA"
  version "$VERSION"
  
  depends_on "go" => :build
  
  def install
    system "make", "build"
    bin.install "build/gif-maker"
  end
  
  test do
    system "#{bin}/gif-maker", "version"
  end
end
EOF

echo "Homebrew formula created at: $FORMULA_PATH"
echo ""
echo "To make this formula available to others:"
echo "1. Create a GitHub release with tag v$VERSION"
echo "2. Upload the tarball ($DIST_DIR/gif-maker-$VERSION.tar.gz) to the release"
echo "3. Either:"
echo "   a. Create your own Homebrew tap repository and add this formula"
echo "   b. Submit the formula to Homebrew/homebrew-core for inclusion"
echo ""
echo "For users to install, they would run:"
echo "  brew install akashdeep/tap/gif-maker"  # If using a tap
echo "  or"
echo "  brew install gif-maker"  # If accepted into homebrew-core
echo ""
echo "Note: You'll need to update the REPO_URL in this script to your actual GitHub repository URL" 