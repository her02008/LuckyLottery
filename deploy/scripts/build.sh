#!/bin/bash

# 多平台构建脚本
set -e

VERSION=${VERSION:-"1.0.0"}
BINARY_NAME="lottery-tool"

echo "Building ${BINARY_NAME} v${VERSION}..."

# 创建输出目录
mkdir -p bin

# 定义构建函数
build_platform() {
    local GOOS=$1
    local GOARCH=$2
    local GOARM=$3
    local SUFFIX=$4

    echo "Building for ${GOOS}/${GOARCH}${GOARM:+v${GOARM}}..."

    local OUTPUT="bin/${BINARY_NAME}-${SUFFIX}"
    
    if [ "$GOOS" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi

    local CGO_ENABLED=1
    local BUILD_FLAGS="-ldflags=-s -w -X main.Version=${VERSION}"

    if [ -n "$GOARM" ]; then
        GOARM=$GOARM CGO_ENABLED=$CGO_ENABLED GOOS=$GOOS GOARCH=$GOARCH go build $BUILD_FLAGS -o ${OUTPUT} ./cmd/cli
    else
        CGO_ENABLED=$CGO_ENABLED GOOS=$GOOS GOARCH=$GOARCH go build $BUILD_FLAGS -o ${OUTPUT} ./cmd/cli
    fi

    echo "  ✓ Built: ${OUTPUT}"
}

# 构建各平台版本
echo ""
echo "=== Building CLI ==="

# Linux
build_platform linux amd64 "" "linux-amd64"
build_platform linux arm64 "" "linux-arm64"
build_platform linux arm 7 "linux-armv7"
build_platform linux arm 6 "linux-armv6"

# macOS
build_platform darwin amd64 "" "darwin-amd64"
build_platform darwin arm64 "" "darwin-arm64"

# Windows
build_platform windows amd64 "" "windows-amd64"

echo ""
echo "=== Building Server ==="

# 构建服务器版本
build_server() {
    local GOOS=$1
    local GOARCH=$2
    local GOARM=$3
    local SUFFIX=$4

    echo "Building server for ${GOOS}/${GOARCH}${GOARM:+v${GOARM}}..."

    local OUTPUT="bin/${BINARY_NAME}-server-${SUFFIX}"
    
    if [ "$GOOS" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi

    local CGO_ENABLED=1
    local BUILD_FLAGS="-ldflags=-s -w -X main.Version=${VERSION}"

    if [ -n "$GOARM" ]; then
        GOARM=$GOARM CGO_ENABLED=$CGO_ENABLED GOOS=$GOOS GOARCH=$GOARCH go build $BUILD_FLAGS -o ${OUTPUT} ./cmd/server
    else
        CGO_ENABLED=$CGO_ENABLED GOOS=$GOOS GOARCH=$GOARCH go build $BUILD_FLAGS -o ${OUTPUT} ./cmd/server
    fi

    echo "  ✓ Built: ${OUTPUT}"
}

# Linux
build_server linux amd64 "" "linux-amd64"
build_server linux arm64 "" "linux-arm64"
build_server linux arm 7 "linux-armv7"
build_server linux arm 6 "linux-armv6"

# macOS
build_server darwin amd64 "" "darwin-amd64"
build_server darwin arm64 "" "darwin-arm64"

# Windows
build_server windows amd64 "" "windows-amd64"

echo ""
echo "=== Build Summary ==="
ls -lh bin/

echo ""
echo "Build complete!"
