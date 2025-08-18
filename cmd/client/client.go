package client

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/easzlab/ezft/pkg/client"
	"github.com/easzlab/ezft/pkg/utils"
	"github.com/easzlab/ezft/pkg/utils/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// client subcommand related variables
var (
	clientURL          string
	clientOutput       string
	clientChunkSize    int64
	clientConcurrency  int
	clientRetryCount   int
	clientResume       bool
	clientAutoChunk    bool
	clientShowProgress bool
	clientLogHome      string
	clientLogLevel     string
)

func init() {
	// client subcommand parameters
	ClientCmd.Flags().StringVarP(&clientURL, "url", "u", "", "Download URL (required)")
	ClientCmd.Flags().StringVarP(&clientOutput, "output", "o", "", "Output file path")
	ClientCmd.Flags().StringVarP(&clientLogHome, "log-home", "", "./logs", "Log file home")
	ClientCmd.Flags().StringVarP(&clientLogLevel, "log-level", "", "debug", "Log level")
	ClientCmd.Flags().Int64VarP(&clientChunkSize, "chunk-size", "s", 1024*1024, "Chunk size (bytes)")
	ClientCmd.Flags().IntVarP(&clientConcurrency, "concurrency", "c", 1, "Concurrency count")
	ClientCmd.Flags().IntVarP(&clientRetryCount, "retry", "r", 3, "Retry count")
	ClientCmd.Flags().BoolVar(&clientResume, "resume", true, "Support resume download")
	ClientCmd.Flags().BoolVar(&clientAutoChunk, "auto-chunk", true, "Auto chunking")
	ClientCmd.Flags().BoolVarP(&clientShowProgress, "progress", "p", true, "Show download progress")

	// Mark required parameters
	ClientCmd.MarkFlagRequired("url")
}

var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "EZFT Client - Download files",
	Long:  "EZFT client supports high-performance concurrent downloads, with resume download, multi-threaded download and progress display.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if clientOutput == "" {
			urlParts := strings.Split(clientURL, "/")
			clientOutput = "down/" + urlParts[len(urlParts)-1]
		}

		if err := utils.EnsureDir(clientLogHome); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Create logger
		l, err := logger.NewLogger(clientLogHome+"/client.log", clientLogLevel)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		// Create download configuration
		config := &client.DownloadConfig{
			URL:            clientURL,
			OutputPath:     clientOutput,
			ChunkSize:      clientChunkSize,
			MaxConcurrency: clientConcurrency,
			RetryCount:     clientRetryCount,
			EnableResume:   clientResume,
			AutoChunk:      clientAutoChunk,
		}

		// Create client
		downloadClient := client.NewClient(config)
		downloadClient.SetLogger(l)

		// Set signal handling
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Println("\nReceived interrupt signal, stopping download...")
			cancel()
		}()

		startTime := time.Now()

		// Start progress display
		if clientShowProgress {
			go downloadClient.ShowProgressLoop(ctx)
		}

		// Execute download
		if err := downloadClient.Download(ctx); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		duration := time.Since(startTime)

		// Display file information
		if info, err := os.Stat(clientOutput); err == nil {
			fmt.Printf("\n✓ Download completed! Duration: %s File size: %s Average speed: %s\n",
				utils.FormatDuration(duration),
				utils.FormatBytes(info.Size()),
				utils.CalculateSpeed(info.Size(), duration),
			)
			l.Info("",
				zap.String("msg", "Download completed"),
				zap.String("duration", utils.FormatDuration(duration)),
				zap.String("file_size", utils.FormatBytes(info.Size())),
				zap.String("average_speed", utils.CalculateSpeed(info.Size(), duration)),
			)
		}

		return nil
	},
}
