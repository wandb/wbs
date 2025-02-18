#!/usr/bin/env bash

# Define GitHub repo
GITHUB_REPO="wandb/wsm"

# Fetch the latest release tag from GitHub
API_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
RELEASE_TAG=$(curl -s $API_URL | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$RELEASE_TAG" ]; then
    echo "Failed to fetch the latest release tag. Exiting."
    exit 1
fi

# Detect OS and architecture
OS=$(uname | tr '[:upper:]' '[:lower:]')
OS="${OS^}" # Capitalize the first letter of OS
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="x86_64";;
    i386) ARCH="i386";;
    i686) ARCH="i386";;
    arm*) ARCH="arm64";;
    aarch64) ARCH="arm64";;
    *) echo "Unsupported architecture: $ARCH"; exit 1;;
esac

# Construct download URL
FILENAME="wsm_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${RELEASE_TAG}/${FILENAME}"
echo "Download URL: ${DOWNLOAD_URL}"

# Get install directory from first argument, default to /usr/local/bin if not provided
INSTALL_DIR="${1:-/usr/local/bin}"

# Create and use temporary directory
TMP_DIR=$(mktemp -d)

# Download tarzip file
echo "Downloading ${FILENAME}..."
curl -L -o "${TMP_DIR}/${FILENAME}" "${DOWNLOAD_URL}" || { echo "Download failed."; rm -rf "$TMP_DIR"; exit 1; }

# Verify download success
if [ $? -ne 0 ]; then
    echo "Download failed."
    rm -rf "$TMP_DIR"
    exit 1
fi

# Extract the tarzip file
echo "Extracting ${FILENAME}..."
tar -xzf "${FILENAME}" -C "${TMP_DIR}" || { echo "Failed to extract ${FILENAME}. Exiting."; rm -rf "$TMP_DIR"; exit 1; }

# Get install directory from first argument, default to /usr/local/bin if not provided
INSTALL_DIR="${1:-/usr/local/bin}"

# Create directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Install the binary
echo "Moving wsm to $INSTALL_DIR/wsm"
chmod +x "${TMP_DIR}/wsm"
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMP_DIR}/wsm" "$INSTALL_DIR/wsm"
else
    sudo mv "${TMP_DIR}/wsm" "$INSTALL_DIR/wsm"
fi

# Clean up
rm -rf "$TMP_DIR"

echo "WSM installed successfully to $INSTALL_DIR/wsm"