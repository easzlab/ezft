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

// Chunk 表示一个下载分片
type Chunk struct {
	Index int64
	Start int64
	End   int64
}

// downloadChunk 下载单个分片
func (c *Client) downloadChunk(ctx context.Context, file *os.File, chunk Chunk) error {
	for retry := 0; retry <= c.config.RetryCount; retry++ {
		if err := c.downloadChunkOnce(ctx, file, chunk); err != nil {
			if retry == c.config.RetryCount {
				return err
			}

			// 等待后重试
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

// downloadChunkOnce 执行一次分片下载
func (c *Client) downloadChunkOnce(ctx context.Context, file *os.File, chunk Chunk) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.URL, nil)
	if err != nil {
		return err
	}

	// 设置Range头
	rangeHeader := fmt.Sprintf("bytes=%d-%d", chunk.Start, chunk.End)
	req.Header.Set("Range", rangeHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("服务器不支持Range请求，状态码: %d", resp.StatusCode)
	}

	// 流式下载：使用缓冲区分批读取和写入
	buffer := make([]byte, 32*1024) // 32KB 缓冲区
	currentOffset := chunk.Start

	for {
		// 检查context是否被取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 读取数据到缓冲区
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			// 确保不超出分片边界
			if currentOffset+int64(n) > chunk.End+1 {
				n = int(chunk.End + 1 - currentOffset)
			}

			// 写入数据到指定位置
			_, writeErr := file.WriteAt(buffer[:n], currentOffset)
			if writeErr != nil {
				return fmt.Errorf("写入数据失败: %w", writeErr)
			}

			currentOffset += int64(n)
		}

		// 检查是否读取完成
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取响应数据失败: %w", err)
		}

		// 检查是否已达到分片结束位置
		if currentOffset > chunk.End {
			break
		}
	}

	return nil
}

// calculateChunks 计算下载分片
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

		// 确保不超出文件边界
		if chunk.End >= end {
			chunk.End = end - 1
		}

		chunks = append(chunks, chunk)
	}

	return chunks
}

// loadFailedChunks 加载失败分片记录
func (c *Client) loadFailedChunks() ([]Chunk, error) {
	var failedChunks []Chunk
	if _, err := os.Stat(c.config.FailedChunksJason); err == nil {
		data, err := os.ReadFile(c.config.FailedChunksJason)
		if err != nil {
			return nil, fmt.Errorf("读取失败分片记录文件失败: %w", err)
		}

		if err := json.Unmarshal(data, &failedChunks); err != nil {
			return nil, fmt.Errorf("解析失败分片记录文件失败: %w", err)
		}
	}

	return failedChunks, nil
}

// saveFailedChunks 保存失败分片记录
func (c *Client) saveFailedChunks(chunks []Chunk) error {
	data, err := json.Marshal(chunks)
	if err != nil {
		return fmt.Errorf("序列化失败分片记录失败: %w", err)
	}

	return os.WriteFile(c.config.FailedChunksJason, data, 0644)
}

// 根据文件大小动态调整分片大小
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
