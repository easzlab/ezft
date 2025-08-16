package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// getOptimalBufferSize returns optimal buffer size based on chunk size configuration
func (c *Client) getOptimalBufferSize() int64 {
	bufSize := c.config.ChunkSize
	if bufSize < 64*1024 {
		bufSize = 64 * 1024 // 64KB minimum
	}
	if bufSize > 2*1024*1024 {
		bufSize = 2 * 1024 * 1024 // 2MB maximum
	}
	return bufSize
}

// BasicDownload downloads the entire file with performance optimizations
func (c *Client) BasicDownload(ctx context.Context) error {
	var lastErr error

	// Retry mechanism
	for attempt := 0; attempt <= c.config.RetryCount; attempt++ {
		if attempt > 0 {
			c.logger.Info("",
				zap.String("msg", fmt.Sprintf("Retry attempt %d/%d", attempt, c.config.RetryCount)),
			)
			// Exponential backoff
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := c.performBasicDownload(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		c.logger.Info("",
			zap.String("msg", fmt.Sprintf("Download attempt %d failed", attempt+1)),
			zap.Error(err),
		)
	}

	return fmt.Errorf("download failed after %d attempts: %w", c.config.RetryCount+1, lastErr)
}

// performBasicDownload performs the actual download with optimizations
func (c *Client) performBasicDownload(ctx context.Context) error {
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

	// Use buffered writer for better performance with unified buffer size
	bufferSize := c.getOptimalBufferSize()
	bufferedWriter := bufio.NewWriterSize(file, int(bufferSize))
	defer func() {
		if flushErr := bufferedWriter.Flush(); flushErr != nil {
			c.logger.Error("",
				zap.String("msg", "failed to flush buffer"),
				zap.Error(flushErr),
			)
		}
	}()

	// Copy data with optimized buffer size
	written, err := c.CopyWithOptimizedBuffer(ctx, bufferedWriter, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	c.logger.Info("",
		zap.String("msg", fmt.Sprintf("Download completed: %d bytes written", written)),
	)
	return nil
}

// CopyWithOptimizedBuffer copies data with optimized buffer size
func (c *Client) CopyWithOptimizedBuffer(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	// Use unified buffer size for consistency
	bufSize := c.getOptimalBufferSize()
	buf := make([]byte, bufSize)
	var written int64

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = fmt.Errorf("invalid write result")
				}
			}
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if er != io.EOF {
				return written, er
			}
			break
		}
	}

	return written, nil
}
