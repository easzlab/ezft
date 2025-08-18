#!/bin/bash

# EZFT End-to-End Test Script
# Tests basic download, non-concurrent chunked download, and concurrent chunked download

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVER_PORT=18080
SERVER_DIR="./temp/server_files"
CLIENT_OUTPUT_DIR="./temp/downloads"
LOG_DIR="./logs"
BINARY_PATH="./build/ezft"
TEST_FILE_NAME="test_file.txt"
TEST_FILE_SIZE="10MB"

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to cleanup
cleanup() {
    print_info "Cleaning up..."
    
    # Kill server process if running
    if [[ -n "$SERVER_PID" ]]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        print_info "Server process stopped"
    fi
    
    # Clean up test files
    rm -rf "$CLIENT_OUTPUT_DIR"
    print_info "Cleanup completed"
}

# Set trap for cleanup on exit
trap cleanup EXIT INT TERM

# Function to wait for server to be ready
wait_for_server() {
    local max_attempts=30
    local attempt=1
    
    print_info "Waiting for server to be ready..."
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -s -f "http://localhost:$SERVER_PORT/" >/dev/null 2>&1; then
            print_success "Server is ready!"
            return 0
        fi
        
        print_info "Attempt $attempt/$max_attempts - Server not ready yet, waiting..."
        sleep 1
        ((attempt++))
    done
    
    print_error "Server failed to start within $max_attempts seconds"
    return 1
}

# Function to create test file
create_test_file() {
    print_info "Creating test file: $TEST_FILE_NAME ($TEST_FILE_SIZE)"
    
    # Ensure server directory exists
    mkdir -p "$SERVER_DIR"
    
    # Create a test file with specified size
    case $TEST_FILE_SIZE in
        "10MB")
            dd if=/dev/zero of="$SERVER_DIR/$TEST_FILE_NAME" bs=1024 count=10240 2>/dev/null
            ;;
        "1MB")
            dd if=/dev/zero of="$SERVER_DIR/$TEST_FILE_NAME" bs=1024 count=1024 2>/dev/null
            ;;
        *)
            # Default to 10MB
            dd if=/dev/zero of="$SERVER_DIR/$TEST_FILE_NAME" bs=1024 count=10240 2>/dev/null
            ;;
    esac
    
    print_success "Test file created: $(ls -lh "$SERVER_DIR/$TEST_FILE_NAME" | awk '{print $5}')"
}

# Function to build binary if not exists
ensure_binary() {
    if [[ ! -f "$BINARY_PATH" ]]; then
        print_info "Binary not found, building..."
        make build
        if [[ ! -f "$BINARY_PATH" ]]; then
            print_error "Failed to build binary"
            exit 1
        fi
        print_success "Binary built successfully"
    else
        print_info "Using existing binary: $BINARY_PATH"
    fi
}

# Function to start server
start_server() {
    print_info "Starting EZFT server on port $SERVER_PORT..."
    print_info "Server directory: $SERVER_DIR"
    
    # Ensure directories exist
    mkdir -p "$SERVER_DIR" "$LOG_DIR"
    
    # Start server in background
    "$BINARY_PATH" server --dir "$SERVER_DIR" --port "$SERVER_PORT" --log-home "$LOG_DIR" --log-level "info" &
    SERVER_PID=$!
    
    print_info "Server started with PID: $SERVER_PID"
    
    # Wait for server to be ready
    if ! wait_for_server; then
        print_error "Failed to start server"
        exit 1
    fi
}

# Function to test download
test_download() {
    local test_name="$1"
    local concurrency="$2"
    local auto_chunk="$3"
    local output_file="$4"
    local enable_resume="$5"
    
    print_info "Testing: $test_name"
    print_info "  Concurrency: $concurrency"
    print_info "  Auto-chunk: $auto_chunk"
    print_info "  Output: $output_file"
    
    # Ensure output directory exists
    mkdir -p "$(dirname "$output_file")"
    
    # Remove existing file if exists
    rm -f "$output_file"
    
    # Build download command
    local url="http://localhost:$SERVER_PORT/$TEST_FILE_NAME"
    local cmd="$BINARY_PATH client --url $url --output $output_file --concurrency $concurrency --auto-chunk $auto_chunk --resume=$enable_resume --progress false --log-home $LOG_DIR --log-level info"
    
    print_info "Executing: $cmd"
    
    # Execute download with timeout
    if timeout 60s $cmd; then
        # Verify file exists and has correct size
        if [[ -f "$output_file" ]]; then
            local original_size=$(stat -f%z "$SERVER_DIR/$TEST_FILE_NAME" 2>/dev/null || stat -c%s "$SERVER_DIR/$TEST_FILE_NAME" 2>/dev/null)
            local downloaded_size=$(stat -f%z "$output_file" 2>/dev/null || stat -c%s "$output_file" 2>/dev/null)
            
            if [[ "$original_size" == "$downloaded_size" ]]; then
                print_success "$test_name completed successfully!"
                print_info "  File size: $(ls -lh "$output_file" | awk '{print $5}')"
                return 0
            else
                print_error "$test_name failed - size mismatch (original: $original_size, downloaded: $downloaded_size)"
                return 1
            fi
        else
            print_error "$test_name failed - output file not found"
            return 1
        fi
    else
        print_error "$test_name failed - download command failed or timed out"
        return 1
    fi
}

# Main execution
main() {
    print_info "Starting EZFT End-to-End Tests"
    print_info "================================"
    
    # Ensure binary exists
    ensure_binary
    
    # Create test file
    create_test_file
    
    # Start server
    start_server
    
    # Test results
    local test_results=()
    
    print_info ""
    print_info "Running download tests..."
    print_info "========================"
    
    # Test 1: Basic download (no concurrency, no auto-chunk)
    if test_download "Basic Download" 1 false "$CLIENT_OUTPUT_DIR/basic_download.txt" false; then
        test_results+=("✓ Basic Download")
    else
        test_results+=("✗ Basic Download")
    fi
    
    sleep 2
    
    # Test 2: Non-concurrent chunked download (no concurrency, with auto-chunk)
    if test_download "Non-Concurrent Chunked Download" 1 true "$CLIENT_OUTPUT_DIR/chunked_download.txt" true; then
        test_results+=("✓ Non-Concurrent Chunked Download")
    else
        test_results+=("✗ Non-Concurrent Chunked Download")
    fi
    
    sleep 2
    
    # Test 3: Concurrent chunked download (with concurrency and auto-chunk)
    if test_download "Concurrent Chunked Download" 4 true "$CLIENT_OUTPUT_DIR/concurrent_download.txt" true; then
        test_results+=("✓ Concurrent Chunked Download")
    else
        test_results+=("✗ Concurrent Chunked Download")
    fi
    
    # Print results summary
    print_info ""
    print_info "All Tests Passed!"
    print_info "===================="
}

# Run main function
main "$@"