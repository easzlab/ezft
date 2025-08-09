package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Chunk represents a download chunk
type Chunk struct {
	Index int64
	Start int64
	End   int64
}

// downloadChunk downloads a single chunk
func (c *Client) downloadChunk(ctx context.Context, file *os.File, chunk Chunk) error {
	for retry := 0; retry <= c.config.RetryCount; retry++ {
		if err := c.downloadChunkOnce(ctx, file, chunk); err != nil {
			if retry == c.config.RetryCount {
				return err
			}

			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(retry+1) * time.Second):
				continue
			}
		}
		return nil
	}
	return nil
}

// downloadChunkOnce executes one chunk download
func (c *Client) downloadChunkOnce(ctx context.Context, file *os.File, chunk Chunk) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.URL, nil)
	if err != nil {
		return err
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ezft/1.0)")

	// Set Range header
	rangeHeader := fmt.Sprintf("bytes=%d-%d", chunk.Start, chunk.End)
	req.Header.Set("Range", rangeHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server does not support Range requests, status code: %d", resp.StatusCode)
	}

	// Streaming download: use buffer for batch read and write
	buffer := make([]byte, 32*1024) // 32KB buffer
	currentOffset := chunk.Start

	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read data to buffer
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			// Ensure not exceeding chunk boundary
			if currentOffset+int64(n) > chunk.End+1 {
				n = int(chunk.End + 1 - currentOffset)
			}

			// Write data to specified position
			_, writeErr := file.WriteAt(buffer[:n], currentOffset)
			if writeErr != nil {
				return fmt.Errorf("failed to write data: %w", writeErr)
			}

			currentOffset += int64(n)
		}

		// Check if reading is complete
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read response data: %w", err)
		}

		// Check if reached chunk end position
		if currentOffset > chunk.End {
			break
		}
	}

	return nil
}

// calculateChunks calculates download chunks
func (c *Client) calculateChunks(start, end int64) []Chunk {
	var chunks []Chunk

	if c.config.AutoChunk {
		c.config.ChunkSize = calculateChunkSize(end - start)
	}

	chunkSize := c.config.ChunkSize

	for i := start; i < end; i += chunkSize {
		chunk := Chunk{
			Index: (i - start) / chunkSize,
			Start: i,
			End:   i + chunkSize - 1,
		}

		// Ensure not exceeding file boundary
		if chunk.End >= end {
			chunk.End = end - 1
		}

		chunks = append(chunks, chunk)
	}

	return chunks
}

// loadFailedChunks loads failed chunks record
func (c *Client) loadFailedChunks() ([]Chunk, error) {
	var failedChunks []Chunk
	if _, err := os.Stat(c.config.FailedChunksJason); err == nil {
		data, err := os.ReadFile(c.config.FailedChunksJason)
		if err != nil {
			return nil, fmt.Errorf("failed to read failed chunks record file: %w", err)
		}

		if err := json.Unmarshal(data, &failedChunks); err != nil {
			return nil, fmt.Errorf("failed to parse failed chunks record file: %w", err)
		}
	}

	return failedChunks, nil
}

// saveFailedChunks saves failed chunks record
func (c *Client) saveFailedChunks(chunks []Chunk) error {
	data, err := json.Marshal(chunks)
	if err != nil {
		return fmt.Errorf("failed to serialize failed chunks record: %w", err)
	}

	return os.WriteFile(c.config.FailedChunksJason, data, 0644)
}

// Dynamically adjust chunk size based on file size
func calculateChunkSize(totalSize int64) int64 {
	switch {
	case totalSize > 100*1024*1024*1024: // >100GB
		return 100 * 1024 * 1024 // 100MB
	case totalSize > 10*1024*1024*1024: // >10GB
		return 50 * 1024 * 1024 // 50MB
	case totalSize > 1*1024*1024*1024: // >1GB
		return 20 * 1024 * 1024 // 20MB
	case totalSize > 100*1024*1024: // >100MB
		return 10 * 1024 * 1024 // 10MB
	default:
		return 4 * 1024 * 1024 // 4MB
	}
}
