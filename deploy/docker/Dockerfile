# 多阶段构建 - 支持多平台
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /app

RUN apk add --no-cache git gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 构建时设置CGO和平台参数
RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    $(if [ "${TARGETARCH}" = "arm" ]; then echo "GOARM=${TARGETVARIANT#v}"; fi) \
    go build -ldflags="-s -w" -o lottery-cli ./cmd/cli && \
    CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    $(if [ "${TARGETARCH}" = "arm" ]; then echo "GOARM=${TARGETVARIANT#v}"; fi) \
    go build -ldflags="-s -w" -o lottery-server ./cmd/server

# 运行阶段
FROM alpine:latest

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

COPY --from=builder /app/lottery-cli /app/
COPY --from=builder /app/lottery-server /app/
COPY config/ /app/config/

RUN mkdir -p /app/data

VOLUME ["/app/data"]

EXPOSE 8080

ENTRYPOINT ["/app/lottery-cli"]
CMD ["--help"]
