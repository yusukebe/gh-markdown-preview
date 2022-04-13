package cmd

import (
	"regexp"
	"strings"
	"testing"
)

func TestTargetFile(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"../testdata/markdown-demo.md", "../testdata/markdown-demo.md"},
		{"../README.md", "../README.md"},
		{"../", "../README.md"},
	}
	for _, tt := range tests {
		actual, err := targetFile(tt.input)
		if err != nil {
			t.Errorf(err.Error())
		}
		expected := tt.expected
		if actual != expected {
			t.Errorf("got %v\n want %v", actual, expected)
		}
	}
	_, err := targetFile("../notfound.md")
	if err == nil {
		t.Errorf("err is nil")
	}
	_, err = targetFile("./")
	if err == nil {
		t.Errorf("err is nil")
	}
}

func TestFindReadme(t *testing.T) {
	actual, _ := findReadme("../")
	expected := "../README.md"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}
	actual, _ = findReadme("../testdata")
	expected = "../testdata/README"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}
	_, err := findReadme("../cmd")
	if err == nil {
		t.Errorf("err is nil")
	}
}

func TestSlurp(t *testing.T) {
	string, err := slurp("../testdata/markdown-demo.md")
	if err != nil {
		t.Errorf(err.Error())
	}
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
	html, err := toHTML(markdown)
	if err != nil {
		t.Errorf(err.Error())
	}
	actual := strings.TrimSpace(html)
	expected := "<p>text</p>"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}
}
