package theme

import "testing"

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Background == "" || cfg.Accent == "" {
		t.Fatalf("theme defaults should be populated")
	}
}
