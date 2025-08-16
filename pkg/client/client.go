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

// Download executes download
func (c *Client) Download(ctx context.Context) error {
	// Get file information
	fileSize, supportsRange, err := c.getFileInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get file information: %w", err)
	}
	log.Printf("file size: %v, supportRange: %v", fileSize, supportsRange)

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
		fmt.Println("Starting resume download")
		// Support resume download, use chunked download
		return c.downloadWithResume(ctx, fileSize)
	}

	// Basic download, no concurrency, no resume support
	fmt.Println("Starting whole file download")
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
		return 0, false, fmt.Errorf("Range request failed: %w", err)
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

// GetProgress gets download progress
func (c *Client) GetProgress() (float64, error) {
	if c.config.FileSize == 0 {
		// Get target file size
		s, _, err := c.getFileInfo(context.Background())
		if err != nil || s == 0 {
			return 0, err
		}
		c.config.FileSize = s
	}

	// Get current downloaded size
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

			// Simple progress bar display
			barWidth := 50
			filled := int(progress * float64(barWidth) / 100)
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

			fmt.Printf("\rDownload progress: [%s] %.1f%%", bar, progress)
		}
	}
}
