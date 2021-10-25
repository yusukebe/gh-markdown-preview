package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh markdown-preview",
	Short: "GitHub CLI extension to preview Markdown",
	Run: func(cmd *cobra.Command, args []string) {

		filename := ""
		if len(args) > 0 {
			filename = args[0]
		}

		port, _ := cmd.Flags().GetInt("port")
		server := Server{port: port}

		reload, _ := cmd.Flags().GetBool("reload")
		server.Serve(filename, reload)

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
	rootCmd.Flags().BoolP("reload", "r", false, "Enable live reloading")
}
