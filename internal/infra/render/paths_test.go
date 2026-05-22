package render

import "testing"

func TestRelativeToRootFromPublicationPage(t *testing.T) {
	got := relativeToRoot("riichirocks/kuikae/")
	if got != "../../" {
		t.Fatalf("unexpected root path: %q", got)
	}
}

func TestRelativeFromPublicationPageToNextPage(t *testing.T) {
	got := relativeFrom("riichirocks/second-post/", "riichirocks/second-post-2/")
	if got != "../second-post-2" {
		t.Fatalf("unexpected relative path: %q", got)
	}
}

func TestRelativeFromTopicIndexToTopicAsset(t *testing.T) {
	got := relativeFrom("riichirocks/index.html", "riichirocks/assets/meta/top_banner.png")
	if got != "assets/meta/top_banner.png" {
		t.Fatalf("unexpected relative path: %q", got)
	}
}
