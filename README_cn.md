# EZFT - 简易文件传输工具

[![Go Version](https://img.shields.io/badge/Go-1.24.4-blue.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/easzlab/ezft)](https://goreportcard.com/report/github.com/easzlab/ezft)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

EZFT (Easy File Transfer) 是一个用 Go 语言编写的高性能文件传输工具，支持客户端下载和服务端功能。具有并发下载、断点续传、进度跟踪和高效文件服务等特性。

## 功能特性

### 客户端功能
- **高性能并发下载**: 支持多线程下载，可配置并发数
- **断点续传支持**: 自动从中断点恢复下载
- **进度跟踪**: 实时下载进度显示，带可视化进度条
- **自动分块**: 智能计算块大小以获得最佳性能
- **重试机制**: 可配置的失败重试次数
- **Range 请求支持**: 高效的部分内容下载
- **信号处理**: 优雅的中断处理 (Ctrl+C)

### 服务端功能
- **高性能文件服务器**: 基于 HTTP 的高效文件服务
- **Range 请求支持**: 支持部分内容传输，用于断点续传
- **多客户端支持**: 并发客户端处理
- **日志中间件**: 请求日志记录和监控
- **目录管理**: 自动目录创建和管理

## 安装

### 从源码构建
```bash
git clone https://github.com/easzlab/ezft.git
cd ezft
go build -o build/ezft cmd/main.go
```

### 使用 Go Install
```bash
go install github.com/easzlab/ezft@latest
```

## 使用方法

### 服务器模式

启动文件服务器来提供目录中的文件：

```bash
# 在默认端口 8080 启动服务器，服务当前目录
./ezft server

# 在自定义端口启动服务器，服务指定目录
./ezft server --port 9000 --dir /path/to/files

# 简短形式
./ezft server -p 9000 -d /path/to/files
```

**服务器选项:**
- `--port, -p`: 服务器端口 (默认: 8080)
- `--dir, -d`: 要服务的根目录 (默认: 当前目录)

### 客户端模式

高性能下载文件，支持断点续传：

```bash
# 基本下载
./ezft client --url http://example.com/file.zip

# 下载到自定义输出路径
./ezft client --url http://example.com/file.zip --output /path/to/save/file.zip

# 高性能并发下载
./ezft client --url http://example.com/file.zip --concurrency 8 --chunk-size 2097152

# 使用自定义设置下载
./ezft client \
  --url http://example.com/file.zip \
  --output downloads/file.zip \
  --concurrency 4 \
  --chunk-size 1048576 \
  --retry 5 \
  --progress
```

**客户端选项:**
- `--url, -u`: 下载 URL (必需)
- `--output, -o`: 输出文件路径 (默认: down/filename)
- `--concurrency, -c`: 并发连接数 (默认: 1)
- `--chunk-size, -s`: 块大小，单位字节 (默认: 1048576 = 1MB)
- `--retry, -r`: 失败重试次数 (默认: 3)
- `--resume`: 启用断点续传 (默认: true)
- `--auto-chunk`: 启用自动块大小计算 (默认: true)
- `--progress, -p`: 显示下载进度 (默认: true)

### 全局选项

```bash
# 显示版本信息
./ezft --version

# 显示帮助
./ezft --help
./ezft client --help
./ezft server --help
```

## 使用示例

### 示例 1: 基本文件服务器
```bash
# 在端口 8080 启动服务器，服务 /var/www/files 目录
./ezft server --dir /var/www/files --port 8080
```

### 示例 2: 高性能下载
```bash
# 使用 8 个并发连接下载大文件
./ezft client \
  --url http://localhost:8080/largefile.iso \
  --concurrency 8 \
  --chunk-size 2097152 \
  --output downloads/largefile.iso
```

### 示例 3: 恢复中断的下载
```bash
# 如果下载被中断，只需再次运行相同的命令
# EZFT 会自动检测并从上次位置恢复
./ezft client --url http://example.com/file.zip --output file.zip
```

## 核心组件

1. **CLI 框架**: 使用 Cobra 构建强大的命令行界面
2. **HTTP 客户端**: 自定义 HTTP 客户端，具有超时和连接管理
3. **并发下载**: 基于 Goroutine 的并发块下载
4. **断点续传逻辑**: 基于 JSON 的失败块跟踪和恢复
5. **进度显示**: 实时进度条和速度计算
6. **文件服务器**: 基于 HTTP 的文件服务器，支持 Range 请求

## 性能特性

- **并发下载**: 多个 goroutine 同时下载不同的块
- **智能分块**: 基于文件大小自动计算块大小
- **连接池**: 高效的 HTTP 连接重用
- **内存高效**: 流式下载，不将整个文件加载到内存
- **断点续传能力**: 失败块跟踪和自动恢复

## 系统要求

- Go 1.24.4 或更高版本
- 网络连接用于下载
- 足够的磁盘空间存储下载文件

## 贡献

1. Fork 仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 支持

如果您遇到任何问题或有疑问，请在 [GitHub 仓库](https://github.com/easzlab/ezft/issues) 上提交 issue。