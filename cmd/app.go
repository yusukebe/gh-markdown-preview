package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/cli/safeexec"
)

func targetFile(filename string) string {
	if filename == "" {
		filename = "."
	}
	info, err := os.Stat(filename)
	if err == nil {
		if info.IsDir() {
			readme := findReadme(filename)
			if readme != "" {
				return readme
			}
		}
	}
	return filename
}

func findReadme(dir string) string {
	files, _ := ioutil.ReadDir(dir)
	for _, f := range files {
		r := regexp.MustCompile(`(?i)^readme`)
		if r.MatchString(f.Name()) {
			return filepath.Join(dir, f.Name())

		}
	}
	return ""
}

func toHTML(markdown string) string {
	sout, _, err := gh("api", "-X", "POST", "/markdown", "-f", fmt.Sprintf("text=%s", markdown))
	if err != nil {
		log.Fatalf("Error:%v", err)
	}
	return sout.String()
}

func slurp(fileName string) string {
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("error:%v", err)
	}
	defer f.Close()
	b, _ := ioutil.ReadAll(f)
	text := string(b)
	return text
}

func gh(args ...string) (sout, eout bytes.Buffer, err error) {
	ghBin, err := safeexec.LookPath("gh")
	if err != nil {
		err = fmt.Errorf("could not find gh. Is it installed? error: %w", err)
		return
	}

	cmd := exec.Command(ghBin, args...)
	cmd.Stderr = &eout
	cmd.Stdout = &sout

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("failed to run gh. error: %w, stderr: %s", err, eout.String())
		return
	}

	return
}

func logInfo(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func logDebug(format string, v ...interface{}) {
	if verbose {
		log.Printf(format, v...)
	}
}
