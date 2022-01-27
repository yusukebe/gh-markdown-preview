package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version string

var verbose = false

type Param struct {
	filename       string
	reload         bool
	forceLightMode bool
	forceDarkMode  bool
}

var rootCmd = &cobra.Command{
	Use:   "gh markdown-preview",
	Short: "GitHub CLI extension to preview Markdown",
	Run: func(cmd *cobra.Command, args []string) {

		showVerionFlag, _ := cmd.Flags().GetBool("version")
		if showVerionFlag {
			showVersion()
			os.Exit(0)
		}

		filename := ""
		if len(args) > 0 {
			filename = args[0]
		}

		verbose, _ = cmd.Flags().GetBool("verbose")

		port, _ := cmd.Flags().GetInt("port")
		server := Server{port: port}

		disableReload, _ := cmd.Flags().GetBool("disable-reload")
		reload := true
		if disableReload {
			reload = false
		}

		forceLightMode, _ := cmd.Flags().GetBool("light-mode")
		forceDarkMode, _ := cmd.Flags().GetBool("dark-mode")

		param := &Param{
			filename:       filename,
			reload:         reload,
			forceLightMode: forceLightMode,
			forceDarkMode:  forceDarkMode,
		}

		server.Serve(param)
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
	rootCmd.Flags().BoolP("version", "", false, "Show the version")
	rootCmd.Flags().BoolP("disable-reload", "", false, "Disable live reloading")
	rootCmd.Flags().BoolP("verbose", "", false, "Show verbose output")
	rootCmd.Flags().BoolP("light-mode", "", false, "Force light mode")
	rootCmd.Flags().BoolP("dark-mode", "", false, "Force dark mode")
}

func showVersion() {
	fmt.Printf("gh-markdown-preview version %s\n", Version)
}
