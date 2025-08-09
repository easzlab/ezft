package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// basicDownload downloads the entire file
func (c *Client) basicDownload(ctx context.Context) error {
	log.Println("Starting whole file download")
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.URL, nil)
	if err != nil {
		return err
	}
	// Set User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ezft/1.0)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed, status code: %d", resp.StatusCode)
	}

	// Create directory
	if err := os.MkdirAll(filepath.Dir(c.config.OutputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create or overwrite file
	flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC

	file, err := os.OpenFile(c.config.OutputPath, flag, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data with progress display
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
