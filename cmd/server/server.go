package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/easzlab/ezft/pkg/server"
	"github.com/spf13/cobra"
)

// server 子命令相关变量
var (
	serverRootDir string
	serverPort    int
)

func init() {
	// server 子命令参数
	ServerCmd.Flags().StringVarP(&serverRootDir, "dir", "d", "./", "文件根目录")
	ServerCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "服务端口")
}

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "EZFT 服务器 - 提供文件下载服务",
	Long:  "EZFT 服务器是一个高性能的文件下载服务器，支持断点续传、Range请求和多客户端并发下载。",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查根目录是否存在，如果不存在则创建
		if err := ensureDir(serverRootDir); err != nil {
			log.Fatalf("创建根目录失败: %v", err)
		}

		// 创建并启动服务器
		srv := server.NewServer(serverRootDir, serverPort)

		if err := srv.Start(); err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	},
}

func ensureDir(dir string) error {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return err
		}
		log.Printf("创建目录: %s", absPath)
	}

	return nil
}
