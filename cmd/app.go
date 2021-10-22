package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/cli/safeexec"
)

func toHTML(markdown string) string {
	sout, _, err := gh("api", "-X", "POST", "/markdown", "-f", fmt.Sprintf("text=%s", markdown))
	if err != nil {
		log.Fatalf("error:%v", err)
	}
	return sout.String()
}

func slurp(fileName string) string {
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("error:%v", err)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
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
