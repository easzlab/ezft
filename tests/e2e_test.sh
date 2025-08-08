#!/bin/bash

# EZFT 端到端测试脚本
# 此脚本测试完整的文件下载流程，包括服务端启动、客户端下载、断点续传等功能

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试配置
SERVER_PORT=18080
SERVER_ROOT="./test_files"
SERVER_PID=""
TEST_FILE="test_large_file.bin"
TEST_FILE_SIZE=10485760  # 10MB
DOWNLOAD_OUTPUT="./downloaded_file.bin"

# 清理函数
cleanup() {
    echo -e "${YELLOW}清理测试环境...${NC}"
    
    # 停止服务端
    if [ ! -z "$SERVER_PID" ]; then
        echo "停止服务端进程 $SERVER_PID"
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    
    # 清理测试文件
    rm -rf "$SERVER_ROOT"
    rm -f "$DOWNLOAD_OUTPUT"
    rm -f "${DOWNLOAD_OUTPUT}.partial"
    
    echo -e "${GREEN}清理完成${NC}"
}

# 设置信号处理
trap cleanup EXIT INT TERM

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# 检查必需的二进制文件
check_binaries() {
    log_info "检查二进制文件..."
    
    if [ ! -f "./build/ezft-server" ]; then
        log_error "找不到 ezft-server 二进制文件，请先运行 make build"
        exit 1
    fi
    
    if [ ! -f "./build/ezft-client" ]; then
        log_error "找不到 ezft-client 二进制文件，请先运行 make build"
        exit 1
    fi
    
    log_info "二进制文件检查通过"
}

# 创建测试文件
create_test_file() {
    log_info "创建测试环境..."
    
    # 创建服务端根目录
    mkdir -p "$SERVER_ROOT"
    
    # 创建测试文件 (10MB)
    log_info "创建 ${TEST_FILE_SIZE} 字节的测试文件..."
    dd if=/dev/urandom of="${SERVER_ROOT}/${TEST_FILE}" bs=1024 count=$((TEST_FILE_SIZE/1024)) 2>/dev/null
    
    # 计算文件MD5用于验证
    TEST_FILE_MD5=$(md5sum "${SERVER_ROOT}/${TEST_FILE}" | cut -d' ' -f1)
    log_info "测试文件MD5: $TEST_FILE_MD5"
}

# 启动服务端
start_server() {
    log_info "启动服务端..."
    
    # 启动服务端在后台
    ./build/ezft-server -r "$SERVER_ROOT" -p $SERVER_PORT &
    SERVER_PID=$!
    
    # 等待服务端启动
    log_info "等待服务端启动..."
    for i in {1..10}; do
        if curl -s "http://localhost:${SERVER_PORT}/health" >/dev/null 2>&1; then
            log_info "服务端启动成功 (PID: $SERVER_PID)"
            return 0
        fi
        sleep 1
    done
    
    log_error "服务端启动失败"
    return 1
}

# 测试完整下载
test_full_download() {
    log_info "测试完整文件下载..."
    
    # 清理之前的下载文件
    rm -f "$DOWNLOAD_OUTPUT"
    
    # 执行下载
    ./build/ezft-client \
        --url "http://localhost:${SERVER_PORT}/download/${TEST_FILE}" \
        --output "$DOWNLOAD_OUTPUT" \
        --chunk-size 1048576 \
        --concurrency 4 \
        --progress=false
    
    # 验证文件是否下载成功
    if [ ! -f "$DOWNLOAD_OUTPUT" ]; then
        log_error "下载文件不存在"
        return 1
    fi
    
    # 验证文件大小
    DOWNLOADED_SIZE=$(stat -f%z "$DOWNLOAD_OUTPUT" 2>/dev/null || stat -c%s "$DOWNLOAD_OUTPUT")
    if [ "$DOWNLOADED_SIZE" != "$TEST_FILE_SIZE" ]; then
        log_error "文件大小不匹配: 期望 $TEST_FILE_SIZE, 实际 $DOWNLOADED_SIZE"
        return 1
    fi
    
    # 验证文件内容 (MD5)
    DOWNLOADED_MD5=$(md5sum "$DOWNLOAD_OUTPUT" | cut -d' ' -f1)
    if [ "$DOWNLOADED_MD5" != "$TEST_FILE_MD5" ]; then
        log_error "文件MD5不匹配: 期望 $TEST_FILE_MD5, 实际 $DOWNLOADED_MD5"
        return 1
    fi
    
    log_info "完整下载测试通过"
}

# 测试断点续传
test_resume_download() {
    log_info "测试断点续传功能..."
    
    # 清理之前的下载文件
    rm -f "$DOWNLOAD_OUTPUT"
    
    # 创建部分下载文件 (下载前一半)
    PARTIAL_SIZE=$((TEST_FILE_SIZE / 2))
    dd if="${SERVER_ROOT}/${TEST_FILE}" of="$DOWNLOAD_OUTPUT" bs=1 count=$PARTIAL_SIZE 2>/dev/null
    
    log_info "创建了 $PARTIAL_SIZE 字节的部分文件，测试续传..."
    
    # 执行断点续传
    ./build/ezft-client \
        --url "http://localhost:${SERVER_PORT}/download/${TEST_FILE}" \
        --output "$DOWNLOAD_OUTPUT" \
        --chunk-size 1048576 \
        --concurrency 2 \
        --progress=false
    
    # 验证文件大小
    DOWNLOADED_SIZE=$(stat -f%z "$DOWNLOAD_OUTPUT" 2>/dev/null || stat -c%s "$DOWNLOAD_OUTPUT")
    if [ "$DOWNLOADED_SIZE" != "$TEST_FILE_SIZE" ]; then
        log_error "续传后文件大小不匹配: 期望 $TEST_FILE_SIZE, 实际 $DOWNLOADED_SIZE"
        return 1
    fi
    
    # 验证文件内容 (MD5)
    DOWNLOADED_MD5=$(md5sum "$DOWNLOAD_OUTPUT" | cut -d' ' -f1)
    if [ "$DOWNLOADED_MD5" != "$TEST_FILE_MD5" ]; then
        log_error "续传后文件MD5不匹配: 期望 $TEST_FILE_MD5, 实际 $DOWNLOADED_MD5"
        return 1
    fi
    
    log_info "断点续传测试通过"
}

# 测试并发下载
test_concurrent_download() {
    log_info "测试高并发下载..."
    
    # 清理之前的下载文件
    rm -f "$DOWNLOAD_OUTPUT"
    
    # 使用高并发数下载
    ./build/ezft-client \
        --url "http://localhost:${SERVER_PORT}/download/${TEST_FILE}" \
        --output "$DOWNLOAD_OUTPUT" \
        --chunk-size 512000 \
        --concurrency 8 \
        --progress=false
    
    # 验证文件内容
    DOWNLOADED_MD5=$(md5sum "$DOWNLOAD_OUTPUT" | cut -d' ' -f1)
    if [ "$DOWNLOADED_MD5" != "$TEST_FILE_MD5" ]; then
        log_error "高并发下载文件MD5不匹配: 期望 $TEST_FILE_MD5, 实际 $DOWNLOADED_MD5"
        return 1
    fi
    
    log_info "高并发下载测试通过"
}

# 测试错误处理
test_error_handling() {
    log_info "测试错误处理..."
    
    # 测试下载不存在的文件
    if ./build/ezft-client \
        --url "http://localhost:${SERVER_PORT}/download/nonexistent.file" \
        --output "./nonexistent_download.bin" \
        --progress=false 2>/dev/null; then
        log_error "下载不存在的文件应该失败"
        return 1
    fi
    
    # 测试无效的URL
    if ./build/ezft-client \
        --url "http://invalid-host:99999/download/file" \
        --output "./invalid_download.bin" \
        --progress=false 2>/dev/null; then
        log_error "无效URL下载应该失败"
        return 1
    fi
    
    log_info "错误处理测试通过"
}

# 性能测试
test_performance() {
    log_info "执行性能测试..."
    
    # 清理之前的下载文件
    rm -f "$DOWNLOAD_OUTPUT"
    
    # 记录开始时间
    START_TIME=$(date +%s)
    
    # 执行下载
    ./build/ezft-client \
        --url "http://localhost:${SERVER_PORT}/download/${TEST_FILE}" \
        --output "$DOWNLOAD_OUTPUT" \
        --chunk-size 2097152 \
        --concurrency 6 \
        --progress=false
    
    # 记录结束时间
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    # 计算速度
    SPEED_MBPS=$(echo "scale=2; $TEST_FILE_SIZE / 1024 / 1024 / $DURATION" | bc -l)
    
    log_info "性能测试结果:"
    log_info "  文件大小: $(echo "scale=2; $TEST_FILE_SIZE / 1024 / 1024" | bc -l) MB"
    log_info "  下载时间: ${DURATION} 秒"
    log_info "  平均速度: ${SPEED_MBPS} MB/s"
    
    # 性能基准 (至少1MB/s)
    MIN_SPEED=1
    if [ $(echo "$SPEED_MBPS >= $MIN_SPEED" | bc -l) -eq 1 ]; then
        log_info "性能测试通过 (速度 >= ${MIN_SPEED} MB/s)"
    else
        log_warn "性能测试警告: 速度较慢 (${SPEED_MBPS} MB/s < ${MIN_SPEED} MB/s)"
    fi
}

# 主测试流程
main() {
    echo -e "${GREEN}=== EZFT 端到端测试开始 ===${NC}"
    
    # 检查依赖
    check_binaries
    
    # 创建测试环境
    create_test_file
    
    # 启动服务端
    start_server
    
    # 执行测试用例
    test_full_download
    test_resume_download
    test_concurrent_download
    test_error_handling
    test_performance
    
    echo -e "${GREEN}=== 所有测试通过! ===${NC}"
}

# 执行主流程
main "$@"