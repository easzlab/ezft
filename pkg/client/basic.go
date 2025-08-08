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

// basicDownload 下载整个文件
func (c *Client) basicDownload(ctx context.Context) error {
	log.Println("开始下载整个文件")
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.URL, nil)
	if err != nil {
		return err
	}
	// 设置User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ezft/1.0)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(c.config.OutputPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 创建或覆盖文件
	flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC

	file, err := os.OpenFile(c.config.OutputPath, flag, 0644)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 使用进度显示的方式复制数据
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}
