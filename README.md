# EZFT - High-Performance File Download Program

EZFT (Easy File Transfer) is a high-performance file download program implemented in Go, supporting both client and server functionality with features like resume download and concurrent downloading.

## Features

✅ **Client and Server** - Provides both file download server and download client  
✅ **Resume Download** - Supports HTTP Range requests, can continue download from interruption point  
✅ **Concurrent Download** - Single download supports multiple goroutines for concurrent processing  
✅ **High Performance** - Optimized for large files, uses chunked download and streaming transfer  
✅ **Comprehensive Testing** - Includes comprehensive unit tests, integration tests and benchmark tests  

## Project Structure

```
ezft/
├── cmd/
│   ├── server/main.go          # Server executable program
│   └── client/main.go          # Client executable program
├── pkg/
│   ├── server/
│   │   ├── server.go           # Server core logic
│   │   └── server_test.go      # Server tests
│   └── client/
│       ├── client.go           # Client core logic
│       └── client_test.go      # Client tests
├── internal/
│   └── utils/
│       ├── utils.go            # Utility functions
│       └── utils_test.go       # Utility function tests
├── go.mod
└── README.md
```

## Installation and Usage

### Prerequisites

- Go 1.20 or higher

### Build Project

```bash
# Clone project
git clone <repository-url>
cd ezft

# Download dependencies
go mod tidy

# Build server
go build -o bin/ezft-server ./cmd/server

# Build client
go build -o bin/ezft-client ./cmd/client
```

### Run Server

```bash
# Start server with default configuration
./bin/ezft-server

# Custom configuration
./bin/ezft-server -root ./files -port 9000

# View help
./bin/ezft-server -help
```

After server starts, it will provide the following interfaces on the specified port:
- `GET /download/<file-path>` - Download file (supports Range requests)
- `GET /info/<file-path>` - Get file information
- `GET /health` - Health check

### Run Client

```bash
# Basic download
./bin/ezft-client -url http://localhost:8080/download/file.zip -output ./file.zip

# Custom configuration
./bin/ezft-client \
  -url http://localhost:8080/download/large-file.bin \
  -output ./large-file.bin \
  -chunk-size 2097152 \
  -concurrency 8 \
  -timeout 60s

# Disable resume download
./bin/ezft-client -url http://localhost:8080/download/file.zip -output ./file.zip -no-resume

# View help
./bin/ezft-client -help
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./pkg/server
go test ./pkg/client
go test ./internal/utils

# Run benchmark tests
go test -bench=. ./...

# View test coverage
go test -cover ./...
```

## Core Technical Features

### Server Features

- **HTTP Range Support**: Full support for HTTP Range request specification
- **Security Control**: Prevents path traversal attacks, secure file access control
- **File Information API**: Provides file size, modification time and other information queries
- **High Concurrency**: Supports multiple clients downloading simultaneously

### Client Features

- **Smart Chunking**: Automatically calculates optimal chunking strategy based on file size
- **Concurrent Download**: Multiple goroutines download different chunks concurrently
- **Resume Download**: Automatically detects downloaded parts, continues from breakpoint
- **Retry Mechanism**: Built-in retry logic improves download success rate
- **Progress Display**: Real-time display of download progress and speed

### Performance Optimization

- **Streaming Transfer**: Avoids large file memory usage
- **Chunked Download**: Splits large files into small blocks for concurrent download
- **Connection Reuse**: Efficient HTTP connection management
- **Memory Control**: Reasonable buffer size control

## API Documentation

### Server API

#### Download File
```
GET /download/<file-path>
Headers:
  Range: bytes=<start>-<end> (optional, for resume download)
```

#### Get File Information
```
GET /info/<file-path>
Response: {
  "name": "filename",
  "size": file_size,
  "modified": "modification_time"
}
```

#### Health Check
```
GET /health
Response: {
  "status": "ok",
  "timestamp": "current_time"
}
```

## Configuration Options

### Server Configuration

- `-root`: File root directory (default: ./files)
- `-port`: Service port (default: 8080)

### Client Configuration

- `-url`: Download URL (required)
- `-output`: Output file path (required)
- `-chunk-size`: Chunk size in bytes (default: 1048576 = 1MB)
- `-concurrency`: Concurrency count (default: 4)
- `-timeout`: Timeout duration (default: 30s)
- `-retry`: Retry count (default: 3)
- `-no-resume`: Disable resume download (default: false)
- `-progress`: Show download progress (default: true)

## Usage Examples

### Scenario 1: Large File Download

```bash
# Start server
./bin/ezft-server -root /path/to/files -port 8080

# Client downloads large file with 8 concurrent connections and 2MB chunks
./bin/ezft-client \
  -url http://localhost:8080/download/large-video.mp4 \
  -output ./large-video.mp4 \
  -chunk-size 2097152 \
  -concurrency 8
```

### Scenario 2: Resume Download

```bash
# First download (may be interrupted)
./bin/ezft-client -url http://localhost:8080/download/file.zip -output ./file.zip

# Second download (automatically continues from breakpoint)
./bin/ezft-client -url http://localhost:8080/download/file.zip -output ./file.zip
```

## Test Coverage

The project includes comprehensive test suites:

- **Unit Tests**: Cover all core functions and methods
- **Integration Tests**: Test complete interaction between client and server
- **Benchmark Tests**: Performance testing and optimization verification
- **Boundary Tests**: Error handling and exception scenario testing

## Performance Metrics

Performance in test environment:

- **Concurrency Capability**: Supports hundreds of concurrent download connections
- **Large File Processing**: Efficiently handles GB-level files
- **Memory Usage**: Constant memory usage, doesn't grow with file size
- **Network Utilization**: Fully utilizes available bandwidth

## Troubleshooting

### Common Issues

1. **Connection Timeout**: Check network connection and server status
2. **Permission Error**: Ensure file read/write permissions
3. **Port Occupied**: Change server port or stop occupying process
4. **File Not Found**: Check file path and server root directory configuration

### Debug Mode

Enable verbose logging through environment variable:

```bash
export EZFT_DEBUG=1
./bin/ezft-client [options]
```

## License

This project is licensed under the MIT License.

## Contribution

Welcome submit Issue and Pull Request to improve project.