package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// downloadWithResume 使用断点续传下载
func (c *Client) downloadWithResume(ctx context.Context, fileSize int64) error {
	// 创建目录
	if err := os.MkdirAll(filepath.Dir(c.config.OutputPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 打开文件进行写入，使用O_RDWR以支持断点续传
	file, err := os.OpenFile(c.config.OutputPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 读取失败分片记录
	failedChunks, err := c.loadFailedChunks()
	if err != nil {
		return fmt.Errorf("加载失败分片记录失败: %w", err)
	}

	// 下载失败分片
	if len(failedChunks) > 0 {
		if err := c.downloadChunksSequentially(ctx, file, failedChunks); err != nil {
			return err
		}
	}

	// 更新文件实际大小
	newExistingSize, err := c.getExistingFileSize()
	if err != nil {
		return fmt.Errorf("更新文件实际大小失败: %w", err)
	}

	// 重新计算剩余分片
	remainingSize := fileSize - newExistingSize
	if remainingSize <= 0 {
		return nil
	}

	chunks := c.calculateChunks(newExistingSize, fileSize)

	fmt.Printf("开始断点续传，分片数: %d，已下载: %d bytes，剩余: %d bytes\n",
		len(chunks), newExistingSize, remainingSize)

	// 使用顺序下载剩余分片
	if c.config.MaxConcurrency < 2 {
		return c.downloadChunksSequentially(ctx, file, chunks)
	}

	// 使用并发下载剩余分片
	return c.downloadChunksConcurrently(ctx, file, chunks)
}

// downloadChunksSequentially 顺序下载分片
func (c *Client) downloadChunksSequentially(ctx context.Context, file *os.File, chunks []Chunk) error {
	for _, chunk := range chunks {
		if err := c.downloadChunk(ctx, file, chunk); err != nil {
			// 记录失败分片
			c.saveFailedChunks([]Chunk{chunk})
			return err
		}
	}
	// 删除失败分片记录
	if _, err := os.Stat(c.config.FailedChunksJason); err == nil {
		if err := os.Remove(c.config.FailedChunksJason); err != nil {
			return fmt.Errorf("删除失败分片记录文件失败: %w", err)
		}
	}
	return nil
}
