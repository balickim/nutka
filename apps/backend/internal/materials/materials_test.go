package materials

import (
	"strings"
	"testing"
)

func TestSanitizeBodyRemovesActiveContentAndKeepsFormatting(t *testing.T) {
	got := SanitizeBody(`<h3>Gamy</h3><p onclick="x()"><strong>C-dur</strong> <script>alert(1)</script><img src=x onerror=alert(1)><a href="javascript:alert(1)">zły</a> <a href="https://example.test/a">nuty</a></p><ul><li>raz</li></ul>`)
	for _, forbidden := range []string{"script", "onclick", "onerror", "<img", "javascript:"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("sanitized body keeps %q: %s", forbidden, got)
		}
	}
	for _, kept := range []string{"<h3>Gamy</h3>", "<strong>C-dur</strong>", `href="https://example.test/a"`, `rel="noreferrer noopener"`, "<li>raz</li>"} {
		if !strings.Contains(got, kept) {
			t.Fatalf("sanitized body drops %q: %s", kept, got)
		}
	}
}

func TestSanitizeBodyKeepsEditorLineBreaks(t *testing.T) {
	got := SanitizeBody(`Pierwsza<div>druga</div><div><br></div><section>trzecia</section>`)
	if !strings.Contains(got, "Pierwsza<div>druga</div><div><br></div>") || strings.Contains(got, "drugatrzecia") {
		t.Fatalf("sanitized body joins editor lines: %s", got)
	}
}

func TestValidateRequiresTitleAndContent(t *testing.T) {
	if SanitizeBody("<p><br></p>&nbsp;") != "" {
		t.Fatal("a body without visible text must become empty")
	}
	cases := []struct {
		title string
		body  string
		files int
		ok    bool
	}{
		{"Gamy", "<p>tekst</p>", 0, true},
		{"Gamy", "", 1, true},
		{"Gamy", "", 0, false},
		{"  ", "<p>tekst</p>", 1, false},
	}
	for _, item := range cases {
		if err := Validate(item.title, item.body, item.files); (err == nil) != item.ok {
			t.Fatalf("Validate(%q, %q, %d) = %v", item.title, item.body, item.files, err)
		}
	}
}
