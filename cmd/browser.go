package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

func openBrowser(port int) error {
	<-time.After(100 * time.Millisecond)
	url := fmt.Sprintf("http://localhost:%d/", port)
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
