package cmd

import (
	"regexp"
	"strings"
	"testing"
)

func TestFindReadme(t *testing.T) {
	actual := findReadme("../")
	expected := "../README.md"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}
	actual = findReadme("../testdata")
	expected = "../testdata/README"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}
}

func TestSelectFile(t *testing.T) {
	actual := targetFile("../testdata/markdown-demo.md")
	expected := "../testdata/markdown-demo.md"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}

	actual = targetFile("../")
	expected = "../README.md"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}
}

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
