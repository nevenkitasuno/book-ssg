package content

import (
	"fmt"
	"slices"
	"time"
)

func NewSite(topics []Topic, globalAssets []Asset) (Site, error) {
	seen := map[string]struct{}{}
	for _, topic := range topics {
		if _, exists := seen[topic.Slug]; exists {
			return Site{}, fmt.Errorf("duplicate topic slug: %s", topic.Slug)
		}
		seen[topic.Slug] = struct{}{}
	}

	slices.SortFunc(topics, func(a, b Topic) int {
		if a.Config.TopicTitle < b.Config.TopicTitle {
			return -1
		}
		if a.Config.TopicTitle > b.Config.TopicTitle {
			return 1
		}
		return 0
	})

	return Site{Topics: topics, GlobalAssets: globalAssets}, nil
}

func NewTopic(slug string, config TopicConfig, links []TopicLink, pages []TopicPage, publications []Publication, assets TopicAssets) (Topic, error) {
	if slug == "" {
		return Topic{}, fmt.Errorf("topic slug is required")
	}
	if config.TopicTitle == "" {
		return Topic{}, fmt.Errorf("topic %s: topic_title is required", slug)
	}
	if config.TopicHeader == "" {
		return Topic{}, fmt.Errorf("topic %s: topic_header is required", slug)
	}

	seenPubs := map[string]struct{}{}
	for _, pub := range publications {
		if _, exists := seenPubs[pub.Slug]; exists {
			return Topic{}, fmt.Errorf("topic %s: duplicate publication slug %s", slug, pub.Slug)
		}
		seenPubs[pub.Slug] = struct{}{}
	}

	slices.SortFunc(publications, func(a, b Publication) int {
		if a.PublishedAt.Equal(b.PublishedAt) {
			if a.Slug < b.Slug {
				return 1
			}
			if a.Slug > b.Slug {
				return -1
			}
			return 0
		}
		if a.PublishedAt.Before(b.PublishedAt) {
			return 1
		}
		return -1
	})

	return Topic{
		Slug:         slug,
		Config:       config,
		Links:        links,
		Pages:        pages,
		Publications: publications,
		Assets:       assets,
	}, nil
}

func NewPublication(folderName, slug, title string, publishedAt time.Time, tags []Tag, pages []PublicationPage, previewText string, previewAsset *Asset, relatedAssets []Asset) (Publication, error) {
	if folderName == "" {
		return Publication{}, fmt.Errorf("publication folder name is required")
	}
	if slug == "" {
		return Publication{}, fmt.Errorf("publication slug is required")
	}
	if len(pages) == 0 {
		return Publication{}, fmt.Errorf("publication %s: at least one page is required", slug)
	}

	slices.SortFunc(pages, func(a, b PublicationPage) int {
		return a.Number - b.Number
	})
	for idx, page := range pages {
		expected := idx + 1
		if page.Number != expected {
			return Publication{}, fmt.Errorf("publication %s: missing page %d", slug, expected)
		}
	}

	seenTags := map[string]struct{}{}
	deduped := make([]Tag, 0, len(tags))
	for _, tag := range tags {
		if _, exists := seenTags[tag.Slug]; exists {
			continue
		}
		seenTags[tag.Slug] = struct{}{}
		deduped = append(deduped, tag)
	}

	if title == "" {
		title = slug
	}

	return Publication{
		FolderName:    folderName,
		Slug:          slug,
		Title:         title,
		PublishedAt:   publishedAt,
		Tags:          deduped,
		Pages:         pages,
		PreviewText:   previewText,
		PreviewAsset:  previewAsset,
		RelatedAssets: relatedAssets,
	}, nil
}
