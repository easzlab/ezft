package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// DownloadConfig download configuration
type DownloadConfig struct {
	URL               string // Download URL
	OutputPath        string // Output file path
	FailedChunksJason string // Failed chunks record file
	ChunkSize         int64  // Size of each chunk
	FileSize          int64  // Size of file to download
	MaxConcurrency    int    // Maximum concurrency
	RetryCount        int    // Retry count
	EnableResume      bool   // Whether to support resume download
	AutoChunk         bool   // Whether to auto chunk, if true, ignore ChunkSize and auto calculate chunk size
}

// DefaultConfig default configuration
func DefaultConfig() *DownloadConfig {
	return &DownloadConfig{
		ChunkSize:      1024 * 1024, // 1MB
		MaxConcurrency: 1,           // Maximum concurrency
		RetryCount:     3,           // Retry 3 times
		EnableResume:   true,        // Support resume download
	}
}

// Client download client
type Client struct {
	config     *DownloadConfig
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new download client
func NewClient(config *DownloadConfig) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second, // Connection establishment timeout
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 10 * time.Second, // Response header timeout
	}

	// Only set default FailedChunksJason if not already set
	if config.FailedChunksJason == "" {
		config.FailedChunksJason = config.OutputPath + ".failed_chunks.json"
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Transport: transport,
		},
	}
}

func (c *Client) SetLogger(logger *zap.Logger) {
	c.logger = logger
}

// Download executes download
func (c *Client) Download(ctx context.Context) error {
	// Get file information
	fileSize, supportsRange, err := c.getFileInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get file information: %w", err)
	}

	c.config.FileSize = fileSize
	c.logger.Info("",
		zap.String("msg", "retrieve file information"),
		zap.Int64("fileSize", fileSize),
		zap.Bool("supportRange", supportsRange),
	)

	// Check if partial download file already exists
	existingSize, err := c.getExistingFileSize()
	if err != nil {
		return fmt.Errorf("failed to check existing file: %w", err)
	}

	// If file is already completely downloaded
	if existingSize == fileSize {
		fmt.Printf("File already completely downloaded: %s\n", c.config.OutputPath)
		return nil
	}

	// Determine download strategy
	if supportsRange && c.config.EnableResume {
		// Support resume download, use chunked download
		return c.downloadWithResume(ctx, fileSize)
	}

	// Basic download, no concurrency, no resume support
	c.logger.Debug("", zap.String("msg", "Starting basic download"))
	return c.BasicDownload(ctx)
}

// getFileInfo gets file information
func (c *Client) getFileInfo(ctx context.Context) (int64, bool, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", c.config.URL, nil)
	if err != nil {
		return 0, false, err
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ezft/1.0)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("server returned error status: %d", resp.StatusCode)
	}

	// Get file size
	contentLength := resp.Header.Get("Content-Length")
	fileSize, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("unable to parse file size: %s", contentLength)
	}

	c.config.FileSize = fileSize

	// Method 1: Check if Range requests are supported
	acceptRanges := resp.Header.Get("Accept-Ranges")
	if strings.ToLower(acceptRanges) == "bytes" {
		return fileSize, true, nil
	}

	// Method 2: Check if Range requests are supported
	req, err = http.NewRequestWithContext(ctx, "GET", c.config.URL, nil)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Range", "bytes=0-0") // Request first byte
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ezft/1.0)")

	resp2, err := c.httpClient.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("range request failed: %w", err)
	}
	defer resp2.Body.Close()

	// Check if status code is 206
	if resp2.StatusCode == http.StatusPartialContent {
		return fileSize, true, nil
	}

	return fileSize, false, nil
}

// getExistingFileSize gets the size of existing file
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
