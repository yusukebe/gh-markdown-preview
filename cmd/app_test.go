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
	html, err := toHTML(markdown, &Param{})
	if err != nil {
		t.Errorf(err.Error())
	}
	actual := strings.TrimSpace(html)
	expected := "<p>text</p>"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}
}

func TestGfmCheckboxes(t *testing.T) {
	string, err := slurp("../testdata/gfm-checkboxes.md")
	if err != nil {
		t.Errorf(err.Error())
	}
	html, err := toHTML(string, &Param{})
	if err != nil {
		t.Errorf(err.Error())
	}
	actual := strings.TrimSpace(html)

	checkBoxes := 0
	checkedCheckBoxes := 0
	uncheckedCheckBoxes := 0
	for _, line := range strings.Split(actual, "\n") {
		if strings.Contains(line, "<input type=\"checkbox\"") {
			checkBoxes++
			if strings.Contains(line, "checked") {
				checkedCheckBoxes++
			} else {
				uncheckedCheckBoxes++
			}
		}
	}
	if checkBoxes != 2 {
		t.Errorf("got %v checkboxes, want 2", checkBoxes)
	}
	if checkedCheckBoxes != 1 {
		t.Errorf("got %v checked checkboxes, want 1", checkedCheckBoxes)
	}
	if uncheckedCheckBoxes != 1 {
		t.Errorf("got %v unchecked checkboxes, want 1", uncheckedCheckBoxes)
	}
}

func TestGfmAlerts(t *testing.T) {
	string, err := slurp("../testdata/gfm-alerts.md")
	if err != nil {
		t.Errorf(err.Error())
	}
	html, err := toHTML(string, &Param{})
	if err != nil {
		t.Errorf(err.Error())
	}
	actual := strings.TrimSpace(html)

	if strings.Contains(actual, "<blockquote") {
		t.Error("got blockquote tag instead of alerts")
	}
}
