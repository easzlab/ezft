package main

import (
	"fmt"
	"os"

	"github.com/easzlab/ezft/cmd/client"
	"github.com/easzlab/ezft/cmd/server"
	"github.com/easzlab/ezft/internal/config"
	"github.com/spf13/cobra"
)

// 版本信息变量
var showVersion bool

func init() {
	// 添加版本标志到根命令
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "显示版本信息")

	// 将子命令添加到根命令
	rootCmd.AddCommand(client.ClientCmd)
	rootCmd.AddCommand(server.ServerCmd)
}

var rootCmd = &cobra.Command{
	Use:   "ezft",
	Short: "EZFT 高性能文件传输工具",
	Long:  "EZFT (Easy File Transfer) 是一个高性能的文件传输工具，支持客户端下载和服务器功能。",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(config.FullVersion())
			return nil
		}
		return cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
