#!/bin/bash

# Usage:
# ./build.sh linux
# ./build.sh windows
# ./build.sh mac-amd64
# ./build.sh mac-arm64

set -e

TARGET=$1
APP_NAME="cal"  # Change this to your app's binary name
OUTPUT_DIR="build"

mkdir -p $OUTPUT_DIR

case "$TARGET" in
    windows)
        GOOS=windows GOARCH=amd64 go build -o $OUTPUT_DIR/${APP_NAME}.exe
        ;;
    linux)
        GOOS=linux GOARCH=amd64 go build -o $OUTPUT_DIR/${APP_NAME}
        ;;
    mac-amd64)
        GOOS=darwin GOARCH=amd64 go build -o $OUTPUT_DIR/${APP_NAME}-mac-amd64
        ;;
    mac-arm64)
        GOOS=darwin GOARCH=arm64 go build -o $OUTPUT_DIR/${APP_NAME}-mac-arm64
        ;;
    *)
        echo "Usage: $0 {windows|linux|mac-amd64|mac-arm64}"
        exit 1
        ;;
esac
 