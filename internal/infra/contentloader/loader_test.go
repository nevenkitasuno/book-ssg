package contentloader

import (
	"strings"
	"testing"
)

func TestLoadSite(t *testing.T) {
	site, err := Load("../../../examples/content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(site.Topics) == 0 {
		t.Fatalf("expected at least one topic")
	}

	var topicFound bool
	var topic = site.Topics[0]
	for _, candidate := range site.Topics {
		if candidate.Slug == "riichi-mahjong" {
			topic = candidate
			topicFound = true
			break
		}
	}
	if !topicFound {
		t.Fatalf("expected topic riichi-mahjong to be present")
	}

	if topic.Config.TopicTitle == "" {
		t.Fatalf("expected topic title to be populated")
	}
	if len(topic.Publications) == 0 {
		t.Fatalf("expected at least one publication")
	}
	found := false
	for _, publication := range topic.Publications {
		if publication.Slug == "mahjong-post" && publication.Title == "Пост про маджонг" {
			if len(publication.Pages) == 0 {
				t.Fatalf("expected publication pages")
			}
			if publication.Pages[0].Markdown == "" {
				t.Fatalf("expected cleaned first page markdown")
			}
			if strings.Contains(publication.Pages[0].Markdown, "# Пост про маджонг") {
				t.Fatalf("expected first heading to be removed from page body")
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected publication mahjong-post to be present")
	}

	pageFound := false
	for _, page := range topic.Pages {
		if page.Slug == "some-page" {
			if page.Title != "Служебная страница" {
				t.Fatalf("unexpected topic page title: %q", page.Title)
			}
			if strings.Contains(page.Markdown, "# Служебная страница") {
				t.Fatalf("expected first heading to be removed from topic page body")
			}
			pageFound = true
			break
		}
	}
	if !pageFound {
		t.Fatalf("expected topic page some-page to be present")
	}
}
