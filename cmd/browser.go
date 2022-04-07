package cmd

import (
	"os/exec"
	"runtime"
	"time"
)

func openBrowser(url string) error {
	<-time.After(100 * time.Millisecond)
	var args []string
	var cmd string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
