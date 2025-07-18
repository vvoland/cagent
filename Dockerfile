# syntax=docker/dockerfile:1

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.6.1 AS xx

# osxcross contains the MacOSX cross toolchain for xx
FROM crazymax/osxcross:14.5-r0-debian AS osxcross

# TODO(platform?)
FROM --platform=$BUILDPLATFORM node:24-alpine@sha256:820e86612c21d0636580206d802a726f2595366e1b867e564cbc652024151e8a AS build-web
WORKDIR /web
COPY web ./
RUN --mount=type=cache,target=/root/.npm \
    npm install && npm run build

FROM golang:1.24-alpine@sha256:daae04ebad0c21149979cd8e9db38f565ecefd8547cf4a591240dc1972cf1399 AS build-agent
RUN apk add --no-cache build-base
WORKDIR /app
COPY . ./
COPY --from=build-web /web/dist ./web/dist
RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o /agent .

FROM alpine:3.22@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
RUN apk add --no-cache curl socat
COPY --from=build-agent /agent /
ENTRYPOINT [ "/agent" ]

FROM --platform=$BUILDPLATFORM golang:1.24.1-alpine3.21 AS builder-base
WORKDIR /src
COPY --from=xx / /
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

FROM builder-base AS builder-darwin
RUN apk add clang
COPY . ./
COPY --from=build-web /web/dist ./web/dist
RUN --mount=type=bind,from=osxcross,src=/osxsdk,target=/xx-sdk \
    --mount=type=cache,target=/root/.cache,id=docker-ai-$TARGETPLATFORM \
    --mount=type=cache,target=/go/pkg/mod <<EOT
    set -x
    CGO_ENABLED=1 xx-go build -trimpath -ldflags "-s -w" -o /binaries/cagent-$TARGETOS-$TARGETARCH .
    xx-verify --static /binaries/cagent-darwin-$TARGETARCH
EOT

FROM builder-base AS builder-linux
RUN apk add clang
RUN xx-apk add libx11-dev musl-dev gcc
COPY . ./
COPY --from=build-web /web/dist ./web/dist
RUN --mount=type=cache,target=/root/.cache,id=docker-ai-$TARGETPLATFORM \
    --mount=type=cache,target=/go/pkg/mod <<EOT
    set -x
    CGO_ENABLED=1 xx-go build -trimpath -ldflags "-s -w -linkmode=external -extldflags '-static'" -o /binaries/cagent-$TARGETOS-$TARGETARCH .
    xx-verify --static /binaries/cagent-linux-$TARGETARCH
EOT

FROM builder-base AS builder-windows
COPY . ./
COPY --from=build-web /web/dist ./web/dist
RUN --mount=type=cache,target=/root/.cache,id=docker-ai-$TARGETPLATFORM \
    --mount=type=cache,target=/go/pkg/mod <<EOT
    set -x
    CGO_ENABLED=0 xx-go build -trimpath -ldflags "-s -w" -o /binaries/cagent-$TARGETOS-$TARGETARCH .
    mv /binaries/cagent-$TARGETOS-$TARGETARCH /binaries/cagent-$TARGETOS-$TARGETARCH.exe
    xx-verify --static /binaries/cagent-windows-$TARGETARCH.exe
EOT

FROM builder-$TARGETOS AS builder

FROM scratch AS cross
COPY --from=builder /binaries . 