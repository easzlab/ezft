package client

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// DownloadConfig 下载配置
type DownloadConfig struct {
	URL               string // 下载URL
	OutputPath        string // 输出文件路径
	FailedChunksJason string // 失败分片记录文件
	ChunkSize         int64  // 每个分片大小
	FileSize          int64  // 待下载文件大小
	MaxConcurrency    int    // 最大并发数
	RetryCount        int    // 重试次数
	EnableResume      bool   // 是否支持断点续传
	AutoChunk         bool   // 是否自动分片，如果为true，则忽略 ChunkSize，自动计算分片大小
}

// DefaultConfig 默认配置
func DefaultConfig() *DownloadConfig {
	return &DownloadConfig{
		ChunkSize:      1024 * 1024, // 1MB
		MaxConcurrency: 1,           // 最大并发
		RetryCount:     3,           // 重试3次
		EnableResume:   true,        // 支持断点续传
	}
}

// Client 下载客户端
type Client struct {
	config     *DownloadConfig
	httpClient *http.Client
}

// NewClient 创建新的下载客户端
func NewClient(config *DownloadConfig) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second, // 连接建立超时
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 10 * time.Second, // 响应头超时
	}

	config.FailedChunksJason = config.OutputPath + ".failed_chunks.json"

	return &Client{
		config: config,
		httpClient: &http.Client{
			Transport: transport,
		},
	}
}

// Download 执行下载
func (c *Client) Download(ctx context.Context) error {
	// 获取文件信息
	fileSize, supportsRange, err := c.getFileInfo(ctx)
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}
	log.Printf("file size: %v, supportRange: %v", fileSize, supportsRange)

	// 检查是否已存在部分下载的文件
	existingSize, err := c.getExistingFileSize()
	if err != nil {
		return fmt.Errorf("检查现有文件失败: %w", err)
	}

	// 如果文件已经完整下载
	if existingSize == fileSize {
		fmt.Printf("文件已经完整下载: %s\n", c.config.OutputPath)
		return nil
	}

	// 确定下载策略
	if supportsRange && c.config.EnableResume {
		fmt.Println("开始断点续传")
		// 支持断点续传，使用分片下载
		return c.downloadWithResume(ctx, fileSize)
	}

	// 基础下载，不支持并发，不支持续传
	fmt.Println("开始下载整个文件")
	return c.basicDownload(ctx)
}

// getFileInfo 获取文件信息
func (c *Client) getFileInfo(ctx context.Context) (int64, bool, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", c.config.URL, nil)
	if err != nil {
		return 0, false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("服务器返回错误状态: %d", resp.StatusCode)
	}

	// 获取文件大小
	contentLength := resp.Header.Get("Content-Length")
	fileSize, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("无法解析文件大小: %s", contentLength)
	}

	c.config.FileSize = fileSize

	// 方法1: 检查是否支持Range请求
	acceptRanges := resp.Header.Get("Accept-Ranges")
	if strings.ToLower(acceptRanges) == "bytes" {
		return fileSize, true, nil
	}

	// 方法2: 检查是否支持Range请求
	req, err = http.NewRequestWithContext(ctx, "GET", c.config.URL, nil)
	if err != nil {
		return 0, false, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Range", "bytes=0-0") // 请求第一个字节
	resp2, err := c.httpClient.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("Range请求失败: %w", err)
	}
	defer resp2.Body.Close()

	// 检查状态码是否为 206
	if resp2.StatusCode == http.StatusPartialContent {
		return fileSize, true, nil
	}

	return fileSize, false, nil
}

// getExistingFileSize 获取已存在文件的大小
func (c *Client) getExistingFileSize() (int64, error) {
	info, err := os.Stat(c.config.OutputPath)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetProgress 获取下载进度
func (c *Client) GetProgress() (float64, error) {
	if c.config.FileSize == 0 {
		// 获取目标文件大小
		s, _, err := c.getFileInfo(context.Background())
		if err != nil || s == 0 {
			return 0, err
		}
		c.config.FileSize = s
	}

	// 获取当前已下载大小
	currentSize, err := c.getExistingFileSize()
	if err != nil {
		return 0, err
	}

	return float64(currentSize) / float64(c.config.FileSize) * 100, nil
}

func (c *Client) ShowProgressLoop(ctx context.Context) {
	time.Sleep(1 * time.Second)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			progress, err := c.GetProgress()
			if err != nil {
				continue
			}

			// 简单的进度条显示
			barWidth := 50
			filled := int(progress * float64(barWidth) / 100)
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

			fmt.Printf("\r下载进度: [%s] %.1f%%", bar, progress)
		}
	}
}
