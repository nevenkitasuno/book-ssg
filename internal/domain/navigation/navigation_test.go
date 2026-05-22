package navigation

import "testing"

func TestPublicationPath(t *testing.T) {
	if got := PublicationPath("riichirocks", "kuikae", 1); got != "riichirocks/kuikae/" {
		t.Fatalf("unexpected path: %s", got)
	}
	if got := PublicationPath("riichirocks", "kuikae", 2); got != "riichirocks/kuikae-2/" {
		t.Fatalf("unexpected path: %s", got)
	}
}
