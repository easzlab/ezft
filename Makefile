# EZFT - Easy File Transfer Makefile

# 变量定义
buildARY_SERVER = build/ezft-server
buildARY_CLIENT = build/ezft-client
GO_FILES = $(shell find . -name "*.go" -type f)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

# 默认目标
.PHONY: all
all: build

# 创建build目录
build:
	mkdir -p build

# 构建所有二进制文件
.PHONY: build
build: build $(buildARY_SERVER) $(buildARY_CLIENT)

# 构建服务端
$(buildARY_SERVER): $(GO_FILES) | build
	go build $(LDFLAGS) -o $(buildARY_SERVER) ./cmd/server

# 构建客户端
$(buildARY_CLIENT): $(GO_FILES) | build
	go build $(LDFLAGS) -o $(buildARY_CLIENT) ./cmd/client

# 单独构建服务端
.PHONY: server
server: $(buildARY_SERVER)

# 单独构建客户端
.PHONY: client
client: $(buildARY_CLIENT)

# 运行测试
.PHONY: test
test:
	go test -v ./...

# 运行测试并显示覆盖率
.PHONY: test-coverage
test-coverage:
	go test -v -cover ./...

# 运行基准测试
.PHONY: bench
bench:
	go test -bench=. ./...

# 代码格式化
.PHONY: fmt
fmt:
	go fmt ./...

# 代码检查
.PHONY: vet
vet:
	go vet ./...

# 下载依赖
.PHONY: deps
deps:
	go mod download
	go mod tidy

# 清理构建产物
.PHONY: clean
clean:
	rm -rf build/
	go clean

# 安装到系统
.PHONY: install
install: build
	cp $(buildARY_SERVER) /usr/local/build/
	cp $(buildARY_CLIENT) /usr/local/build/

# 卸载
.PHONY: uninstall
uninstall:
	rm -f /usr/local/build/ezft-server
	rm -f /usr/local/build/ezft-client

# 交叉编译
.PHONY: build-all
build-all: clean
	# Linux amd64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/ezft-server-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/ezft-client-linux-amd64 ./cmd/client
	# Linux arm64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o build/ezft-server-linux-arm64 ./cmd/server
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o build/ezft-client-linux-arm64 ./cmd/client
	# macOS amd64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/ezft-server-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/ezft-client-darwin-amd64 ./cmd/client
	# macOS arm64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/ezft-server-darwin-arm64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/ezft-client-darwin-arm64 ./cmd/client
	# Windows amd64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/ezft-server-windows-amd64.exe ./cmd/server
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/ezft-client-windows-amd64.exe ./cmd/client

# 运行服务端（开发模式）
.PHONY: run-server
run-server:
	go run ./cmd/server -root ./files -port 8080

# 运行客户端测试下载（需要先启动服务端）
.PHONY: run-client-test
run-client-test:
	@echo "请确保服务端已启动，然后手动运行："
	@echo "go run ./cmd/client -url http://localhost:8080/download/<文件名> -output <输出路径>"

# 完整的检查流程
.PHONY: check
check: fmt vet test

# 发布准备
.PHONY: release
release: check build-all

# 显示帮助信息
.PHONY: help
help:
	@echo "EZFT - Easy File Transfer"
	@echo ""
	@echo "可用目标:"
	@echo "  build         - 构建服务端和客户端二进制文件"
	@echo "  server        - 只构建服务端"
	@echo "  client        - 只构建客户端"
	@echo "  test          - 运行单元测试"
	@echo "  test-coverage - 运行测试并显示覆盖率"
	@echo "  bench         - 运行基准测试"
	@echo "  fmt           - 格式化代码"
	@echo "  vet           - 代码静态检查"
	@echo "  deps          - 下载并整理依赖"
	@echo "  clean         - 清理构建产物"
	@echo "  install       - 安装到系统(/usr/local/build)"
	@echo "  uninstall     - 从系统卸载"
	@echo "  build-all     - 交叉编译所有平台版本"
	@echo "  run-server    - 运行服务端(开发模式)"
	@echo "  check         - 完整检查(fmt+vet+test)"
	@echo "  release       - 发布准备(check+build-all)"
	@echo "  help          - 显示此帮助信息"
	@echo ""
	@echo "示例:"
	@echo "  make build               # 构建项目"
	@echo "  make test               # 运行测试"
	@echo "  make run-server         # 启动服务端"
	@echo "  make install            # 安装到系统"