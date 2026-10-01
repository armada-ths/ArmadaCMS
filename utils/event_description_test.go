package utils

import (
	"strings"
	"testing"
)

func TestSanitizeEventDescriptionPreservesSafeFormatting(t *testing.T) {
	description := `<p>Welcome <strong>students</strong>.</p><ul><li>Bring ID</li></ul><a href="https://example.com/info" title="More information">Read more</a>`

	sanitized := SanitizeEventDescription(&description)
	if sanitized == nil {
		t.Fatal("expected a sanitized description")
	}

	for _, expected := range []string{
		"<p>Welcome <strong>students</strong>.</p>",
		"<ul><li>Bring ID</li></ul>",
		`href="https://example.com/info"`,
		`rel="nofollow noreferrer noopener"`,
		`target="_blank"`,
	} {
		if !strings.Contains(*sanitized, expected) {
			t.Errorf("expected sanitized HTML to contain %q, got %q", expected, *sanitized)
		}
	}
}

func TestSanitizeEventDescriptionRemovesExecutableMarkup(t *testing.T) {
	description := `<p onclick="alert(1)">Safe text</p><img src=x onerror="alert(2)"><script>alert(3)</script><svg onload="alert(4)"></svg><iframe src="https://evil.example"></iframe><a href="javascript:alert(5)">Bad link</a>`

	sanitized := SanitizeEventDescription(&description)
	if sanitized == nil {
		t.Fatal("expected a sanitized description")
	}

	for _, forbidden := range []string{
		"onclick", "<img", "onerror", "<script", "alert(3)",
		"<svg", "onload", "<iframe", "javascript:",
	} {
		if strings.Contains(strings.ToLower(*sanitized), forbidden) {
			t.Errorf("expected %q to be removed, got %q", forbidden, *sanitized)
		}
	}

	if !strings.Contains(*sanitized, "<p>Safe text</p>") {
		t.Errorf("expected safe text to remain, got %q", *sanitized)
	}
}

func TestSanitizeEventDescriptionPreservesNil(t *testing.T) {
	if sanitized := SanitizeEventDescription(nil); sanitized != nil {
		t.Fatalf("expected nil, got %q", *sanitized)
	}
}
