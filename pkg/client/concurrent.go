package client

import (
	"context"
	"fmt"
	"os"
	"sync"
)

// downloadChunksConcurrently downloads chunks concurrently
func (c *Client) downloadChunksConcurrently(ctx context.Context, file *os.File, chunks []Chunk) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(chunks))
	semaphore := make(chan struct{}, c.config.MaxConcurrency)

	// Used to collect failed chunks
	var failedChunksMutex sync.Mutex
	var failedChunks []Chunk

	for _, chunk := range chunks {
		// Control concurrency
		semaphore <- struct{}{}
		wg.Add(1)

		go func(ck Chunk) {
			defer func() {
				wg.Done()
				<-semaphore
			}()

			if err := c.downloadChunk(ctx, file, ck); err != nil {
				// Record failed chunk
				failedChunksMutex.Lock()
				failedChunks = append(failedChunks, ck)
				failedChunksMutex.Unlock()

				// Send error to channel
				errChan <- fmt.Errorf("failed to download chunk %d: %w", ck.Index, err)
			}
		}(chunk)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Close error channel
	close(errChan)

	// Collect all errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// If there are failed chunks, save record
	if len(failedChunks) > 0 {
		if err := c.saveFailedChunks(failedChunks); err != nil {
			return fmt.Errorf("failed to save failed chunks record: %w", err)
		}
	}

	// If there are errors, return the first error
	if len(errors) > 0 {
		return errors[0]
	}

	// All chunks downloaded successfully, delete failed chunks record file
	if _, err := os.Stat(c.config.FailedChunksJason); err == nil {
		if err := os.Remove(c.config.FailedChunksJason); err != nil {
			return fmt.Errorf("failed to delete failed chunks record file: %w", err)
		}
	}

	return nil
}
