#!/usr/bin/env bash
# Build aiusage for all supported platforms.
# Run from the repository root.
set -euo pipefail

VERSION="${1:-1.0.0}"
LDFLAGS="-s -w -X main.version=$VERSION"
OUTDIR="dist"

rm -rf "$OUTDIR"
mkdir -p "$OUTDIR"

PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "windows/arm64"
)

for platform in "${PLATFORMS[@]}"; do
    IFS="/" read -r GOOS GOARCH <<< "$platform"
    output="$OUTDIR/aiusage-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output="$output.exe"
    fi
    echo "Building $GOOS/$GOARCH → $output"
    GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "$LDFLAGS" -o "$output" .
done

echo ""
echo "Built $(ls "$OUTDIR" | wc -l) binaries in $OUTDIR/"
ls -lh "$OUTDIR"/
