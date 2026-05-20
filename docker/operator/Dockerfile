# Build the manager binary
FROM golang:1.21.11 as builder

ENV CGO_ENABLED 0
ENV GOOS linux

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build
RUN go build -ldflags="-s -w" -a -o manager cmd/main.go && echo "Building GOARCH of $GOARCH.."

# Use debian:12-slim as minimal base image to package the manager binary.
# Upgrade libssl3 to fix CVE-2025-15467 (requires >= 3.0.18-1~deb12u2).
FROM debian:12-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends libssl3=3.0.18-1~deb12u2 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /
COPY --from=builder /workspace/manager /bin/manager
USER 65532:65532
