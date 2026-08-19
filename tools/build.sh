#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
OS=${GOOS:-$(uname -s | tr '[:upper:]' '[:lower:]')}
ARCH=${GOARCH:-$(uname -m)}

case "$OS" in
  mingw*|msys*|cygwin*) OS=windows ;;
esac

case "$ARCH" in
  arm64|aarch64) ARCH=arm64 ;;
  x86_64|amd64) ARCH=amd64 ;;
esac

OUT_DIR="$ROOT_DIR/tools/bin-$OS-$ARCH"
mkdir -p "$OUT_DIR"
OUT_NAME="cloudcc"
if [ "$OS" = "windows" ]; then
  OUT_NAME="cloudcc.exe"
fi

BUILD_LDFLAGS=${CLOUDCC_BUILD_LDFLAGS:--s -w}

cd "$ROOT_DIR/cli"
GOOS="$OS" GOARCH="$ARCH" CGO_ENABLED=0 go build -trimpath -ldflags="$BUILD_LDFLAGS" -o "$OUT_DIR/$OUT_NAME" ./cmd/cloudcc
chmod +x "$OUT_DIR/$OUT_NAME"
echo "Built: $OUT_DIR/$OUT_NAME"
