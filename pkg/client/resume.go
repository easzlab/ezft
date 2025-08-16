package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// downloadWithResume downloads using resume functionality
func (c *Client) downloadWithResume(ctx context.Context, fileSize int64) error {
	// Create directory
	if err := os.MkdirAll(filepath.Dir(c.config.OutputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file for writing, use O_RDWR to support resume download
	file, err := os.OpenFile(c.config.OutputPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Load failed chunks record
	failedChunks, err := c.loadFailedChunks()
	if err != nil {
		return fmt.Errorf("failed to load failed chunks record: %w", err)
	}

	// Download failed chunks
	if len(failedChunks) > 0 {
		if err := c.downloadChunksSequentially(ctx, file, failedChunks); err != nil {
			return err
		}
	}

	// Update actual file size
	newExistingSize, err := c.getExistingFileSize()
	if err != nil {
		return fmt.Errorf("failed to update actual file size: %w", err)
	}

	// Recalculate remaining chunks
	remainingSize := fileSize - newExistingSize
	if remainingSize <= 0 {
		return nil
	}

	chunks := c.calculateChunks(newExistingSize, fileSize)

	c.logger.Info("",
		zap.String("msg", "Starting resume download"),
		zap.Int("chunks", len(chunks)),
		zap.Int(("concurrent"), c.config.MaxConcurrency),
		zap.Int64("downloaded", newExistingSize),
		zap.Int64("remaining", remainingSize),
	)

	// Use sequential download for remaining chunks
	if c.config.MaxConcurrency < 2 {
		return c.downloadChunksSequentially(ctx, file, chunks)
	}

	// Use concurrent download for remaining chunks
	return c.downloadChunksConcurrently(ctx, file, chunks)
}

// downloadChunksSequentially downloads chunks sequentially
func (c *Client) downloadChunksSequentially(ctx context.Context, file *os.File, chunks []Chunk) error {
	for _, chunk := range chunks {
		if err := c.downloadChunk(ctx, file, chunk); err != nil {
			// Record failed chunk
			if saveErr := c.saveFailedChunks([]Chunk{chunk}); saveErr != nil {
				// Log the save error but still return the original download error
				c.logger.Info("",
					zap.String("msg", "failed to save failed chunks"),
					zap.Error(saveErr),
				)
			}
			return err
		}
	}
	// Delete failed chunks record after successful completion
	if _, err := os.Stat(c.config.FailedChunksJason); err == nil {
		if err := os.Remove(c.config.FailedChunksJason); err != nil {
			return fmt.Errorf("failed to delete failed chunks record file: %w", err)
		}
	}
	return nil
}
