package server

import (
	"fmt"
	"log"

	"github.com/easzlab/ezft/pkg/server"
	"github.com/easzlab/ezft/pkg/utils"
	"github.com/spf13/cobra"
)

// server subcommand related variables
var (
	serverRootDir string
	serverPort    int
)

func init() {
	// server subcommand parameters
	ServerCmd.Flags().StringVarP(&serverRootDir, "dir", "d", "./", "File root directory")
	ServerCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "Service port")
}

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "EZFT Server - Provide file download service",
	Long:  "EZFT server is a high-performance file download server that supports resume download, Range requests and multi-client concurrent downloads.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if root directory exists, create if it doesn't exist
		if err := utils.EnsureDir(serverRootDir); err != nil {
			log.Fatalf("Failed to create root directory: %v", err)
		}

		// Create and start server
		srv := server.NewServer(serverRootDir, serverPort)

		if err := srv.Start(); err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	},
}
