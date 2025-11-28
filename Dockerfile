# syntax=docker/dockerfile:1

ARG GO_VERSION="1.25.4"
ARG ALPINE_VERSION="3.22"

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.7.0 AS xx

# osxcross contains the MacOSX cross toolchain for xx
FROM crazymax/osxcross:15.5-debian AS osxcross

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder-base
COPY --from=xx / /
WORKDIR /src
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download
ENV CGO_ENABLED=1


ARG GIT_TAG
ARG GIT_COMMIT
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

FROM builder-base AS builder-darwin
RUN apk add clang
COPY . ./
RUN --mount=type=bind,from=osxcross,src=/osxsdk,target=/xx-sdk \
    --mount=type=cache,target=/root/.cache,id=docker-ai-$TARGETPLATFORM \
    --mount=type=cache,target=/go/pkg/mod <<EOT
    set -ex
    xx-go build -trimpath -ldflags "-s -w -X 'github.com/docker/cagent/pkg/version.Version=$GIT_TAG' -X 'github.com/docker/cagent/pkg/version.Commit=$GIT_COMMIT'" -o /binaries/cagent-$TARGETOS-$TARGETARCH .
    xx-verify --static /binaries/cagent-darwin-$TARGETARCH
EOT

FROM builder-base AS builder-linux
RUN apk add clang
RUN xx-apk add musl-dev gcc
COPY . ./
RUN --mount=type=cache,target=/root/.cache,id=docker-ai-$TARGETPLATFORM \
    --mount=type=cache,target=/go/pkg/mod <<EOT
    set -ex
    xx-go build -trimpath -ldflags "-s -w -linkmode=external -extldflags '-static' -X 'github.com/docker/cagent/pkg/version.Version=$GIT_TAG' -X 'github.com/docker/cagent/pkg/version.Commit=$GIT_COMMIT'" -o /binaries/cagent-$TARGETOS-$TARGETARCH .
    xx-verify --static /binaries/cagent-linux-$TARGETARCH
EOT

FROM builder-base AS builder-windows
RUN apk add zig build-base
COPY . ./
RUN --mount=type=cache,target=/root/.cache,id=docker-ai-$TARGETPLATFORM \
    --mount=type=cache,target=/go/pkg/mod <<EOT
    set -ex
    CC="zig cc -target x86_64-windows-gnu" CXX="zig c++ -target x86_64-windows-gnu" xx-go build -trimpath -ldflags "-s -w -X 'github.com/docker/cagent/pkg/version.Version=$GIT_TAG' -X 'github.com/docker/cagent/pkg/version.Commit=$GIT_COMMIT'" -o /binaries/cagent-$TARGETOS-$TARGETARCH .
    mv /binaries/cagent-$TARGETOS-$TARGETARCH /binaries/cagent-$TARGETOS-$TARGETARCH.exe
    xx-verify --static /binaries/cagent-windows-$TARGETARCH.exe
EOT

FROM builder-$TARGETOS AS builder

FROM scratch AS local
ARG TARGETOS TARGETARCH
COPY --from=builder /binaries/cagent-$TARGETOS-$TARGETARCH cagent

FROM scratch AS cross
COPY --from=builder /binaries .

FROM alpine
RUN apk add --no-cache ca-certificates docker-cli
RUN addgroup -S cagent && adduser -S -G cagent cagent
ARG TARGETOS TARGETARCH
ENV DOCKER_MCP_IN_CONTAINER=1
ENV TERM=xterm-256color
RUN mkdir /data /work && chmod 777 /data /work
COPY --from=docker/mcp-gateway:v2 /docker-mcp /usr/local/lib/docker/cli-plugins/
COPY --from=builder /binaries/cagent-$TARGETOS-$TARGETARCH /cagent
USER cagent
WORKDIR /work
ENTRYPOINT ["/cagent"]
