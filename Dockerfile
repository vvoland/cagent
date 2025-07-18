# syntax=docker/dockerfile:1

FROM node:24-alpine@sha256:820e86612c21d0636580206d802a726f2595366e1b867e564cbc652024151e8a AS build-web
WORKDIR /web
COPY web ./
RUN --mount=type=cache,target=/root/.npm \
    npm install && npm run build

FROM golang:1.24-alpine@sha256:daae04ebad0c21149979cd8e9db38f565ecefd8547cf4a591240dc1972cf1399 AS build-agent
WORKDIR /app
COPY . ./
COPY --from=build-web /web/dist ./web/dist
RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /agent .

FROM alpine:3.22@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
RUN apk add --no-cache curl socat
COPY --from=build-agent /agent /
ENTRYPOINT [ "/agent" ]
