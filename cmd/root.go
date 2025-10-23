package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "stream-keeper",
	Short: "Keep YouTube live streams alive by streaming placeholder content",
	Long: `stream-keeper is a CLI tool that keeps YouTube live streams active by streaming
placeholder content when the primary camera goes offline. This prevents YouTube from
cutting the stream and changing the URL.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {}
