# Build go
FROM golang:1.26.5-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
ENV CGO_ENABLED=0
ENV GOEXPERIMENT=jsonv2
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN go build -v -o V2bX \
    -tags "sing xray hysteria2 with_quic with_grpc with_utls with_wireguard with_acme with_gvisor" \
    -trimpath \
    -ldflags "-X 'github.com/InazumaV/V2bX/cmd.version=${VERSION}' -s -w -buildid="

# Release
FROM alpine:3.21
RUN apk --update --no-cache add tzdata ca-certificates \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN mkdir /etc/V2bX/
COPY --from=builder /app/V2bX /usr/local/bin

# Root is kept for host-network low-port binding; restrict ApiKey file perms on the host (chmod 600).
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD pidof V2bX || exit 1

ENTRYPOINT [ "V2bX", "server", "--config", "/etc/V2bX/config.json"]
