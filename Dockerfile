# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache curl make git libc-dev bash file gcc linux-headers eudev-dev
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go install .

FROM alpine:latest
RUN apk add --no-cache jq
COPY --from=builder /go/bin/* /usr/local/bin/
ENTRYPOINT [ "goat-debug" ]
