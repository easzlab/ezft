package client

import (
	"context"
	"fmt"
	"strings"
	"time"
)

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
