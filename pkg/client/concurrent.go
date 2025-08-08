package client

import (
	"context"
	"fmt"
	"os"
	"sync"
)

// downloadChunksConcurrently 并发下载分片
func (c *Client) downloadChunksConcurrently(ctx context.Context, file *os.File, chunks []Chunk) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(chunks))
	semaphore := make(chan struct{}, c.config.MaxConcurrency)

	// 用于收集失败的分片
	var failedChunksMutex sync.Mutex
	var failedChunks []Chunk

	for _, chunk := range chunks {
		// 控制并发数
		semaphore <- struct{}{}
		wg.Add(1)

		go func(ck Chunk) {
			defer func() {
				wg.Done()
				<-semaphore
			}()

			if err := c.downloadChunk(ctx, file, ck); err != nil {
				// 记录失败的分片
				failedChunksMutex.Lock()
				failedChunks = append(failedChunks, ck)
				failedChunksMutex.Unlock()

				// 发送错误到channel
				errChan <- fmt.Errorf("下载分片 %d 失败: %w", ck.Index, err)
			}
		}(chunk)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 关闭错误channel
	close(errChan)

	// 收集所有错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// 如果有失败的分片，保存记录
	if len(failedChunks) > 0 {
		if err := c.saveFailedChunks(failedChunks); err != nil {
			return fmt.Errorf("保存失败分片记录失败: %w", err)
		}
	}

	// 如果有错误，返回第一个错误
	if len(errors) > 0 {
		return errors[0]
	}

	// 所有分片下载成功，删除失败分片记录文件
	if _, err := os.Stat(c.config.FailedChunksJason); err == nil {
		if err := os.Remove(c.config.FailedChunksJason); err != nil {
			return fmt.Errorf("删除失败分片记录文件失败: %w", err)
		}
	}

	return nil
}
