package preview

import "testing"

func TestExtract(t *testing.T) {
	got := Extract("# Заголовок\n\nПервый абзац.\n\nВторой абзац.")
	if got != "Первый абзац." {
		t.Fatalf("unexpected preview: %q", got)
	}
}
