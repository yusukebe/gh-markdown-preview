package cmd

import (
	"regexp"
	"strings"
	"testing"
)

func TestSlurp(t *testing.T) {
	string := slurp("../testdata/markdown-demo.md")
	match := "Headings"
	r := regexp.MustCompile(match)
	if r.MatchString(string) == false {
		t.Errorf("content do not match %v\n", match)
	}
}

func TestGh(t *testing.T) {
	o, _, _ := gh("help")
	match := "USAGE"
	r := regexp.MustCompile(match)
	if r.MatchString(o.String()) == false {
		t.Errorf("content do not match %v\n", match)
	}
}

func TestToHTML(t *testing.T) {
	markdown := "text"
	actual := strings.TrimSpace(toHTML(markdown))
	expected := "<p>text</p>"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}
}
