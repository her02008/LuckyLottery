.PHONY: all build build-cli build-server clean

BINARY_NAME=lottery-cli
SERVER_NAME=lottery-server
DOCKER_IMAGE=lottery-tool:latest

# 默认构建当前平台
all: build

build: build-cli build-server

build-cli:
	CGO_ENABLED=1 go build -o bin/$(BINARY_NAME) ./cmd/cli

build-server:
	CGO_ENABLED=1 go build -o bin/$(SERVER_NAME) ./cmd/server

# 多平台交叉编译
build-all: build-linux-amd64 build-linux-arm64 build-linux-armv7 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

build-linux-amd64:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/cli
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/$(SERVER_NAME)-linux-amd64 ./cmd/server

build-linux-arm64:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/cli
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o bin/$(SERVER_NAME)-linux-arm64 ./cmd/server

build-linux-armv7:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 go build -o bin/$(BINARY_NAME)-linux-armv7 ./cmd/cli
	CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 go build -o bin/$(SERVER_NAME)-linux-armv7 ./cmd/server

build-darwin-amd64:
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/cli
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o bin/$(SERVER_NAME)-darwin-amd64 ./cmd/server

build-darwin-arm64:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/cli
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o bin/$(SERVER_NAME)-darwin-arm64 ./cmd/server

build-windows-amd64:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/cli
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o bin/$(SERVER_NAME)-windows-amd64.exe ./cmd/server

# Docker构建
docker-build:
	docker build -t $(DOCKER_IMAGE) -f deploy/docker/Dockerfile .

docker-run:
	docker run -v $(shell pwd)/data:/app/data -p 8080:8080 $(DOCKER_IMAGE)

# 测试
test:
	go test -v ./...

# 清理
clean:
	rm -rf bin/

# 安装依赖
deps:
	go mod download
	go mod tidy

# 运行开发版本
dev-cli:
	go run ./cmd/cli

dev-server:
	go run ./cmd/server
