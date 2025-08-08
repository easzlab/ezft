# EZFT - 高性能文件下载程序

EZFT (Easy File Transfer) 是一个用 Go 语言实现的高性能文件下载程序，支持客户端和服务端，具有断点续传、并发下载等特性。

## 功能特性

✅ **客户端和服务端** - 同时提供文件下载服务端和下载客户端  
✅ **断点续传** - 支持HTTP Range请求，可从中断处继续下载  
✅ **并发下载** - 单次下载支持多个goroutine并发处理  
✅ **高性能** - 针对大文件优化，使用分片下载和流式传输  
✅ **完善测试** - 包含全面的单元测试、集成测试和基准测试  

## 项目结构

```
ezft/
├── cmd/
│   ├── server/main.go          # 服务端可执行程序
│   └── client/main.go          # 客户端可执行程序
├── pkg/
│   ├── server/
│   │   ├── server.go           # 服务端核心逻辑
│   │   └── server_test.go      # 服务端测试
│   └── client/
│       ├── client.go           # 客户端核心逻辑
│       └── client_test.go      # 客户端测试
├── internal/
│   └── utils/
│       ├── utils.go            # 工具函数
│       └── utils_test.go       # 工具函数测试
├── go.mod
└── README.md
```

## 安装和使用

### 前置要求

- Go 1.20 或更高版本

### 构建项目

```bash
# 克隆项目
git clone <repository-url>
cd ezft

# 下载依赖
go mod tidy

# 构建服务端
go build -o bin/ezft-server ./cmd/server

# 构建客户端
go build -o bin/ezft-client ./cmd/client
```

### 运行服务端

```bash
# 使用默认配置启动服务端
./bin/ezft-server

# 自定义配置
./bin/ezft-server -root ./files -port 9000

# 查看帮助
./bin/ezft-server -help
```

服务端启动后，会在指定端口提供以下接口：
- `GET /download/<文件路径>` - 下载文件（支持Range请求）
- `GET /info/<文件路径>` - 获取文件信息
- `GET /health` - 健康检查

### 运行客户端

```bash
# 基本下载
./bin/ezft-client -url http://localhost:8080/download/file.zip -output ./file.zip

# 自定义配置
./bin/ezft-client \
  -url http://localhost:8080/download/large-file.bin \
  -output ./large-file.bin \
  -chunk-size 2097152 \
  -concurrency 8 \
  -timeout 60s

# 禁用断点续传
./bin/ezft-client -url http://localhost:8080/download/file.zip -output ./file.zip -no-resume

# 查看帮助
./bin/ezft-client -help
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./pkg/server
go test ./pkg/client
go test ./internal/utils

# 运行基准测试
go test -bench=. ./...

# 查看测试覆盖率
go test -cover ./...
```

## 核心技术特性

### 服务端特性

- **HTTP Range支持**: 完整支持HTTP Range请求规范
- **安全控制**: 防止路径遍历攻击，安全的文件访问控制
- **文件信息API**: 提供文件大小、修改时间等信息查询
- **高并发**: 支持多客户端同时下载

### 客户端特性

- **智能分片**: 根据文件大小自动计算最优分片策略
- **并发下载**: 多个goroutine并发下载不同分片
- **断点续传**: 自动检测已下载部分，从断点继续下载
- **重试机制**: 内置重试逻辑，提高下载成功率
- **进度显示**: 实时显示下载进度和速度

### 性能优化

- **流式传输**: 避免大文件内存占用
- **分片下载**: 将大文件分割为小块并发下载
- **连接复用**: 高效的HTTP连接管理
- **内存控制**: 合理的缓冲区大小控制

## API文档

### 服务端API

#### 下载文件
```
GET /download/<文件路径>
Headers:
  Range: bytes=<start>-<end> (可选，用于断点续传)
```

#### 获取文件信息
```
GET /info/<文件路径>
Response: {
  "name": "文件名",
  "size": 文件大小,
  "modified": "修改时间"
}
```

#### 健康检查
```
GET /health
Response: {
  "status": "ok",
  "timestamp": "当前时间"
}
```

## 配置选项

### 服务端配置

- `-root`: 文件根目录 (默认: ./files)
- `-port`: 服务端口 (默认: 8080)

### 客户端配置

- `-url`: 下载URL (必需)
- `-output`: 输出文件路径 (必需)
- `-chunk-size`: 分片大小，字节 (默认: 1048576 = 1MB)
- `-concurrency`: 并发数 (默认: 4)
- `-timeout`: 超时时间 (默认: 30s)
- `-retry`: 重试次数 (默认: 3)
- `-no-resume`: 禁用断点续传 (默认: false)
- `-progress`: 显示下载进度 (默认: true)

## 使用示例

### 场景1: 大文件下载

```bash
# 启动服务端
./bin/ezft-server -root /path/to/files -port 8080

# 客户端下载大文件，使用8个并发，2MB分片
./bin/ezft-client \
  -url http://localhost:8080/download/large-video.mp4 \
  -output ./large-video.mp4 \
  -chunk-size 2097152 \
  -concurrency 8
```

### 场景2: 断点续传

```bash
# 第一次下载（可能中断）
./bin/ezft-client -url http://localhost:8080/download/file.zip -output ./file.zip

# 第二次下载（自动从断点继续）
./bin/ezft-client -url http://localhost:8080/download/file.zip -output ./file.zip
```

## 测试覆盖

项目包含完善的测试套件：

- **单元测试**: 覆盖所有核心函数和方法
- **集成测试**: 测试客户端和服务端的完整交互
- **基准测试**: 性能测试和优化验证
- **边界测试**: 错误处理和异常情况测试

## 性能指标

在测试环境中的性能表现：

- **并发能力**: 支持数百个并发下载连接
- **大文件处理**: 高效处理GB级别文件
- **内存使用**: 恒定内存占用，不随文件大小增长
- **网络利用率**: 充分利用可用带宽

## 故障排除

### 常见问题

1. **连接超时**: 检查网络连接和服务端状态
2. **权限错误**: 确保有文件读写权限
3. **端口占用**: 更换服务端口或停止占用进程
4. **文件不存在**: 检查文件路径和服务端根目录配置

### 调试模式

可以通过环境变量启用详细日志：

```bash
export EZFT_DEBUG=1
./bin/ezft-client [options]
```

## 许可证

本项目采用 MIT 许可证。

## 贡献

欢迎提交 Issue 和 Pull Request 来改进项目。