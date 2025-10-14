#!/bin/bash

set -eu

OUTPUT_DIR="release"
APP_ENTRY="main.go"
APP_NAME="balance_bot"
VERSION="${1:-dev}"
BUILD_DATE="$(date +%Y-%m-%d.%H:%M)"

mkdir -p "$OUTPUT_DIR"

echo "Start building ${APP_NAME} | version: ${VERSION} | time: ${BUILD_DATE}"
echo "Output dir: ./${OUTPUT_DIR}/"

# 使用循环字符串（兼容 sh）
for TARGET in \
    "linux amd64 _linux_amd64" \
    "linux arm64 _linux_arm64" \
    "darwin arm64 _darwin_arm64" \
    "darwin amd64 _darwin_amd64" \
    "windows amd64 _windows_amd64.exe" \
    "windows arm64 _windows_arm64.exe"
do
    # 手动分割字段
    set -- $TARGET
    GOOS="$1"
    GOARCH="$2"
    SUFFIX="$3"

    BINARY_NAME="${APP_NAME}${SUFFIX}"
    OUTPUT_PATH="${OUTPUT_DIR}/${BINARY_NAME}"

    echo "Building: ${GOOS}/${GOARCH} -> ${BINARY_NAME}"

    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
        -trimpath \
        -ldflags "-s -w \
            -X 'main.version=${VERSION}' \
            -X 'main.date=${BUILD_DATE}'" \
        -o "$OUTPUT_PATH" \
        "$APP_ENTRY"

    echo "Build completed: ${OUTPUT_PATH}"
done

echo ""
echo "All Building completed. Binaries are in the '${OUTPUT_DIR}' directory."
