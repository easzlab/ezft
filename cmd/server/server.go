package server

import (
	"fmt"

	"github.com/easzlab/ezft/pkg/server"
	"github.com/easzlab/ezft/pkg/utils"
	"github.com/easzlab/ezft/pkg/utils/logger"
	"github.com/spf13/cobra"
)

// server subcommand related variables
var (
	serverRootDir  string
	serverPort     int
	serverLogHome  string
	serverLogLevel string
)

func init() {
	// server subcommand parameters
	ServerCmd.Flags().StringVarP(&serverRootDir, "dir", "d", "./", "File root directory")
	ServerCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "Service port")
	ServerCmd.Flags().StringVarP(&serverLogHome, "log-home", "", "./logs", "Log file home")
	ServerCmd.Flags().StringVarP(&serverLogLevel, "log-level", "", "debug", "Log level")
}

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "EZFT Server - Provide file download service",
	Long:  "EZFT server is a high-performance file download server that supports resume download, Range requests and multi-client concurrent downloads.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if root directory exists, create if it doesn't exist
		if err := utils.EnsureDir(serverRootDir); err != nil {
			return fmt.Errorf("failed to create root directory: %w", err)
		}

		if err := utils.EnsureDir(serverLogHome); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Create logger
		l, err := logger.NewLogger(serverLogHome+"/server.log", serverLogLevel)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		// Create and start server
		srv := server.NewServer(serverRootDir, serverPort)
		srv.SetLogger(l)

		if err := srv.Start(); err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	},
}
