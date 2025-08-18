# EZFT - Easy File Transfer

[![Go Version](https://img.shields.io/badge/Go-1.24.4-blue.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/easzlab/ezft)](https://goreportcard.com/report/github.com/easzlab/ezft)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[README](README.md) | [中文文档](README_zh.md)

EZFT (Easy File Transfer) is a high-performance file transfer tool written in Go that supports both client download and server functionality. It features concurrent downloads, resume capability, progress tracking, and efficient file serving.

## Features

### Client Features
- **High-Performance Concurrent Downloads**: Multi-threaded downloading with configurable concurrency
- **Resume Download Support**: Automatic resume from interruption points
- **Progress Tracking**: Real-time download progress display with visual progress bar
- **Auto Chunking**: Intelligent chunk size calculation for optimal performance
- **Retry Mechanism**: Configurable retry count for failed downloads
- **Range Request Support**: Efficient partial content downloads
- **Signal Handling**: Graceful interruption handling (Ctrl+C)

### Server Features
- **High-Performance File Server**: Efficient HTTP-based file serving
- **Range Request Support**: Partial content delivery for resume downloads
- **Multi-Client Support**: Concurrent client handling
- **Logging Middleware**: Request logging and monitoring
- **Directory Management**: Automatic directory creation and management

## Installation

### From Source
```bash
git clone https://github.com/easzlab/ezft.git
cd ezft
go build -o build/ezft cmd/main.go
```

### Using Go Install
```bash
go install github.com/easzlab/ezft@latest
```

## Usage

### Server Mode

Start a file server to serve files from a directory:

```bash
# Start server on default port 8080 serving current directory
./ezft server

# Start server on custom port serving specific directory
./ezft server --port 9000 --dir /path/to/files

# Short form
./ezft server -p 9000 -d /path/to/files
```

**Server Options:**
- `--port, -p`: Server port (default: 8080)
- `--dir, -d`: Root directory to serve files from (default: current directory)

### Client Mode

Download files with high performance and resume capability:

```bash
# Basic download
./ezft client --url http://example.com/file.zip

# Download with custom output path
./ezft client --url http://example.com/file.zip --output /path/to/save/file.zip

# High-performance concurrent download
./ezft client --url http://example.com/file.zip --concurrency 8 --chunk-size 2097152

# Download with custom settings
./ezft client \
  --url http://example.com/file.zip \
  --output downloads/file.zip \
  --concurrency 4 \
  --chunk-size 1048576 \
  --retry 5 \
  --progress
```

**Client Options:**
- `--url, -u`: Download URL (required)
- `--output, -o`: Output file path (default: down/filename)
- `--concurrency, -c`: Number of concurrent connections (default: 1)
- `--chunk-size, -s`: Chunk size in bytes (default: 1048576 = 1MB)
- `--retry, -r`: Retry count for failed downloads (default: 3)
- `--resume`: Enable resume download (default: true)
- `--auto-chunk`: Enable automatic chunk size calculation (default: true)
- `--progress, -p`: Show download progress (default: true)

### Global Options

```bash
# Show version information
./ezft --version

# Show help
./ezft --help
./ezft client --help
./ezft server --help
```

## Examples

### Example 1: Basic File Server
```bash
# Start server serving files from /var/www/files on port 8080
./ezft server --dir /var/www/files --port 8080
```

### Example 2: High-Performance Download
```bash
# Download large file with 8 concurrent connections
./ezft client \
  --url http://localhost:8080/largefile.iso \
  --concurrency 8 \
  --chunk-size 2097152 \
  --output downloads/largefile.iso
```

### Example 3: Resume Interrupted Download
```bash
# If download was interrupted, simply run the same command again
# EZFT will automatically detect and resume from the last position
./ezft client --url http://example.com/file.zip --output file.zip
```

## Key Components

1. **CLI Framework**: Built with Cobra for robust command-line interface
2. **HTTP Client**: Custom HTTP client with timeout and connection management
3. **Concurrent Downloads**: Goroutine-based concurrent chunk downloading
4. **Resume Logic**: JSON-based failed chunk tracking and recovery
5. **Progress Display**: Real-time progress bar with speed calculation
6. **File Server**: HTTP-based file server with Range request support

## Performance Features

- **Concurrent Downloads**: Multiple goroutines download different chunks simultaneously
- **Intelligent Chunking**: Automatic chunk size calculation based on file size
- **Connection Pooling**: Efficient HTTP connection reuse
- **Memory Efficient**: Streaming downloads without loading entire files into memory
- **Resume Capability**: Failed chunk tracking and automatic recovery

## Requirements

- Go 1.24.4 or later
- Network connectivity for downloads
- Sufficient disk space for downloaded files

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

If you encounter any issues or have questions, please open an issue on the [GitHub repository](https://github.com/easzlab/ezft/issues).