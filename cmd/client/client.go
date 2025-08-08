package client

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/easzlab/ezft/pkg/client"
	"github.com/spf13/cobra"
)

// client 子命令相关变量
var (
	clientURL          string
	clientOutput       string
	clientChunkSize    int64
	clientConcurrency  int
	clientRetryCount   int
	clientResume       bool
	clientAutoChunk    bool
	clientShowProgress bool
)

func init() {
	// client 子命令参数
	ClientCmd.Flags().StringVarP(&clientURL, "url", "u", "", "下载URL (必需)")
	ClientCmd.Flags().StringVarP(&clientOutput, "output", "o", "", "输出文件路径 (必需)")
	ClientCmd.Flags().Int64VarP(&clientChunkSize, "chunk-size", "s", 1024*1024, "分片大小 (字节)")
	ClientCmd.Flags().IntVarP(&clientConcurrency, "concurrency", "c", 1, "并发数")
	ClientCmd.Flags().IntVarP(&clientRetryCount, "retry", "r", 3, "重试次数")
	ClientCmd.Flags().BoolVar(&clientResume, "resume", true, "支持断点续传")
	ClientCmd.Flags().BoolVar(&clientAutoChunk, "auto-chunk", true, "自动分片")
	ClientCmd.Flags().BoolVarP(&clientShowProgress, "progress", "p", true, "显示下载进度")

	// 标记必需参数
	ClientCmd.MarkFlagRequired("url")
	ClientCmd.MarkFlagRequired("output")
}

var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "EZFT 客户端 - 下载文件",
	Long:  "EZFT 客户端支持高性能并发下载，支持断点续传、多线程下载和进度显示。",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建下载配置
		config := &client.DownloadConfig{
			URL:            clientURL,
			OutputPath:     clientOutput,
			ChunkSize:      clientChunkSize,
			MaxConcurrency: clientConcurrency,
			RetryCount:     clientRetryCount,
			EnableResume:   clientResume,
			AutoChunk:      clientAutoChunk,
		}

		// 创建客户端
		downloadClient := client.NewClient(config)

		// 设置信号处理
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Println("\n接收到中断信号，正在停止下载...")
			cancel()
		}()

		startTime := time.Now()

		// 启动进度显示
		if clientShowProgress {
			go downloadClient.ShowProgressLoop(ctx)
		}

		// 执行下载
		err := downloadClient.Download(ctx)
		if err != nil {
			return fmt.Errorf("下载失败: %w", err)
		}

		duration := time.Since(startTime)
		fmt.Printf("\n✓ 下载完成！耗时: %v\n", duration)

		// 显示文件信息
		if info, err := os.Stat(clientOutput); err == nil {
			fmt.Printf("文件大小: %d bytes (%.2f MB)\n", info.Size(), float64(info.Size())/(1024*1024))
			if duration > 0 {
				speed := float64(info.Size()) / duration.Seconds() / (1024 * 1024)
				fmt.Printf("平均速度: %.2f MB/s\n", speed)
			}
		}

		return nil
	},
}
