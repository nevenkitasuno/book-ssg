package frontmatter

import "testing"

func TestParse(t *testing.T) {
	meta, body, err := Parse("---\ntitle: Куикаэ\ntags:\n  - Риичи\n  - Правила риичи\n---\n\nТекст")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.GetString("title") != "Куикаэ" {
		t.Fatalf("unexpected title: %q", meta.GetString("title"))
	}
	tags := meta.GetStrings("tags")
	if len(tags) != 2 {
		t.Fatalf("unexpected tags len: %d", len(tags))
	}
	if body != "\nТекст" {
		t.Fatalf("unexpected body: %q", body)
	}
}
