#!/bin/bash

set -e

: "${OUTPUT_DIR:=./dist}"

# Build info
: "${VERSION:="$(git describe --tags --exact-match 2>/dev/null || echo dev)"}"
: "${GIT_COMMIT:="$(git rev-parse HEAD 2>/dev/null || echo unknown)"}"
: "${BUILD_TIME:="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"}"

GO_MODULE="github.com/docker/cagent"
LDFLAGS="-s -w \
    -X '${GO_MODULE}/cmd/root.Version=${VERSION}' \
    -X '${GO_MODULE}/cmd/root.Commit=${GIT_COMMIT}' \
    -X '${GO_MODULE}/cmd/root.BuildTime=${BUILD_TIME}'"

echo "Building to $OUTPUT_DIR with flags $LDFLAGS"
mkdir -p "$OUTPUT_DIR"

go build -trimpath -ldflags "$LDFLAGS" -o "${OUTPUT_DIR}/cagent" main.go
