package main

import (
	"fmt"
	"os"

	"github.com/easzlab/ezft/cmd/client"
	"github.com/easzlab/ezft/cmd/server"
	"github.com/easzlab/ezft/internal/config"
	"github.com/spf13/cobra"
)

var showVersion bool

func init() {
	// Add version flag to root command
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")

	// Add subcommands to root command
	rootCmd.AddCommand(client.ClientCmd)
	rootCmd.AddCommand(server.ServerCmd)
}

var rootCmd = &cobra.Command{
	Use:   "ezft",
	Short: "EZFT high-performance file transfer tool",
	Long:  "EZFT (Easy File Transfer) is a high-performance file transfer tool that supports client download and server functionality.",
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
