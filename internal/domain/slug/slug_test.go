package slug

import "testing"

func TestParsePublicationDir(t *testing.T) {
	date, parsedSlug, err := ParsePublicationDir("2026-05-03-kuikae")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := date.Format("2006-01-02"); got != "2026-05-03" {
		t.Fatalf("unexpected date: %s", got)
	}
	if parsedSlug != "kuikae" {
		t.Fatalf("unexpected slug: %s", parsedSlug)
	}
}

func TestNormalizeUnicode(t *testing.T) {
	got := Normalize("Правила риичи")
	if got == "" {
		t.Fatalf("expected non-empty unicode slug")
	}
}
