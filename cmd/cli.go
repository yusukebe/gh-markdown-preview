package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh markdown-preview",
	Short: "Preview Markdown with gh extension",
	Run: func(cmd *cobra.Command, args []string) {

		filename := ""
		if len(args) > 0 {
			filename = args[0]
		}

		port, _ := cmd.Flags().GetInt("port")
		server := Server{port: port}

		server.Serve(filename)

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntP("port", "p", 3333, "TCP port number of this server")
}
