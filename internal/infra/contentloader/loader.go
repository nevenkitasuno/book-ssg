package contentloader

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"blog-ssg/internal/domain/content"
	"blog-ssg/internal/domain/slug"
	"blog-ssg/pkg/assets"
	"blog-ssg/pkg/frontmatter"
	"blog-ssg/pkg/preview"
	"blog-ssg/pkg/theme"
)

func Load(contentRoot string) (content.Site, error) {
	entries, err := os.ReadDir(contentRoot)
	if err != nil {
		return content.Site{}, err
	}

	var topics []content.Topic
	var globalAssets []content.Asset

	for _, entry := range entries {
		fullPath := filepath.Join(contentRoot, entry.Name())
		if !entry.IsDir() {
			globalAssets = append(globalAssets, content.Asset{
				SourcePath: fullPath,
				PublicPath: entry.Name(),
				Kind:       assets.Classify(fullPath),
			})
			continue
		}

		isTopic, err := looksLikeTopic(fullPath)
		if err != nil {
			return content.Site{}, err
		}
		if !isTopic {
			moreAssets, err := collectStaticDirAssets(fullPath, contentRoot)
			if err != nil {
				return content.Site{}, err
			}
			globalAssets = append(globalAssets, moreAssets...)
			continue
		}

		topic, err := loadTopic(fullPath)
		if err != nil {
			return content.Site{}, err
		}
		topics = append(topics, topic)
	}

	return content.NewSite(topics, globalAssets)
}

func looksLikeTopic(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "meta" {
			return true, nil
		}
		if _, _, err := slug.ParsePublicationDir(entry.Name()); err == nil {
			return true, nil
		}
	}
	return false, nil
}

func collectStaticDirAssets(root string, contentRoot string) ([]content.Asset, error) {
	var result []content.Asset
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(contentRoot, path)
		if err != nil {
			return err
		}
		result = append(result, content.Asset{
			SourcePath: path,
			PublicPath: filepath.ToSlash(rel),
			Kind:       assets.Classify(path),
		})
		return nil
	})
	return result, err
}

func loadTopic(topicPath string) (content.Topic, error) {
	topicSlug := slug.Normalize(filepath.Base(topicPath))
	metaPath := filepath.Join(topicPath, "meta")
	configPath := filepath.Join(metaPath, "Config.yaml")

	config, err := theme.LoadConfig(configPath)
	if err != nil {
		return content.Topic{}, fmt.Errorf("topic %s: load config: %w", topicSlug, err)
	}

	links, err := loadTopicLinks(filepath.Join(metaPath, "Links.md"))
	if err != nil {
		return content.Topic{}, fmt.Errorf("topic %s: load links: %w", topicSlug, err)
	}

	topicPages, bannerAsset, otherTopicAssets, err := loadTopicMeta(topicPath, metaPath)
	if err != nil {
		return content.Topic{}, err
	}

	entries, err := os.ReadDir(topicPath)
	if err != nil {
		return content.Topic{}, err
	}

	var publications []content.Publication
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "meta" {
			continue
		}
		publishedAt, pubSlug, err := slug.ParsePublicationDir(entry.Name())
		if err != nil {
			continue
		}
		pub, err := loadPublication(filepath.Join(topicPath, entry.Name()), topicSlug, entry.Name(), pubSlug, publishedAt)
		if err != nil {
			return content.Topic{}, err
		}
		publications = append(publications, pub)
	}

	return content.NewTopic(topicSlug, config, links, topicPages, publications, content.TopicAssets{
		Banner: bannerAsset,
		Other:  otherTopicAssets,
	})
}

func loadTopicLinks(path string) ([]content.TopicLink, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var links []content.TopicLink
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "- ") {
			continue
		}
		link, ok := parseLink(strings.TrimPrefix(line, "- "))
		if ok {
			links = append(links, link)
		}
	}
	return links, scanner.Err()
}

func parseLink(value string) (content.TopicLink, bool) {
	if strings.HasPrefix(value, "[[") && strings.HasSuffix(value, "]]") {
		inner := strings.TrimSuffix(strings.TrimPrefix(value, "[["), "]]")
		parts := strings.SplitN(inner, "|", 2)
		target := strings.TrimSpace(parts[0])
		label := target
		if len(parts) == 2 {
			label = strings.TrimSpace(parts[1])
		}
		return content.TopicLink{Label: label, Target: target, Kind: content.LinkKindInternal}, true
	}
	if strings.HasPrefix(value, "[") {
		closeLabel := strings.Index(value, "](")
		if closeLabel > 0 && strings.HasSuffix(value, ")") {
			label := value[1:closeLabel]
			target := strings.TrimSuffix(value[closeLabel+2:], ")")
			kind := content.LinkKindInternal
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
				kind = content.LinkKindExternal
			}
			return content.TopicLink{Label: label, Target: target, Kind: kind}, true
		}
	}
	return content.TopicLink{}, false
}

func loadTopicMeta(topicPath, metaPath string) ([]content.TopicPage, *content.Asset, []content.Asset, error) {
	info, err := os.Stat(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, nil, fmt.Errorf("topic meta is not a directory: %s", metaPath)
	}

	entries, err := os.ReadDir(metaPath)
	if err != nil {
		return nil, nil, nil, err
	}
	var pages []content.TopicPage
	var banner *content.Asset
	var other []content.Asset

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		fullPath := filepath.Join(metaPath, name)
		switch {
		case name == "Config.yaml" || name == "Links.md":
			continue
		case assets.Classify(fullPath) == content.AssetKindBanner:
			asset := content.Asset{
				SourcePath: fullPath,
				PublicPath: filepath.ToSlash(filepath.Join(filepath.Base(topicPath), "assets", "meta", name)),
				Kind:       content.AssetKindBanner,
			}
			banner = &asset
		case strings.HasSuffix(strings.ToLower(name), ".md"):
			body, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, nil, nil, err
			}
			pageSlug := slug.Normalize(strings.TrimSuffix(name, filepath.Ext(name)))
			title, cleanedBody := extractPublicationTitle(string(body))
			if title == "" {
				title = slug.Humanize(pageSlug)
			}
			pages = append(pages, content.TopicPage{
				Slug:     pageSlug,
				Title:    title,
				Markdown: cleanedBody,
			})
		default:
			other = append(other, content.Asset{
				SourcePath: fullPath,
				PublicPath: filepath.ToSlash(filepath.Join(filepath.Base(topicPath), "assets", "meta", name)),
				Kind:       assets.Classify(fullPath),
			})
		}
	}

	sort.Slice(pages, func(i, j int) bool { return pages[i].Slug < pages[j].Slug })
	return pages, banner, other, nil
}

func loadPublication(pubPath, topicSlug, folderName, pubSlug string, publishedAt time.Time) (content.Publication, error) {
	entries, err := os.ReadDir(pubPath)
	if err != nil {
		return content.Publication{}, err
	}

	var pages []content.PublicationPage
	var relatedAssets []content.Asset
	var previewAsset *content.Asset

	for _, entry := range entries {
		fullPath := filepath.Join(pubPath, entry.Name())
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			number, parseErr := strconv.Atoi(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
			if parseErr != nil {
				continue
			}
			body, readErr := os.ReadFile(fullPath)
			if readErr != nil {
				return content.Publication{}, readErr
			}
			pages = append(pages, content.PublicationPage{
				Number:   number,
				Markdown: string(body),
			})
			continue
		}

		asset := content.Asset{
			SourcePath: fullPath,
			PublicPath: filepath.ToSlash(filepath.Join(topicSlug, "assets", pubSlug, entry.Name())),
			Kind:       assets.Classify(fullPath),
		}
		if asset.Kind == content.AssetKindPreview {
			copy := asset
			previewAsset = &copy
		}
		relatedAssets = append(relatedAssets, asset)
	}

	sort.Slice(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })
	if len(pages) == 0 {
		return content.Publication{}, fmt.Errorf("publication %s/%s: no markdown pages found", topicSlug, folderName)
	}

	firstMeta, firstBody, err := frontmatter.Parse(pages[0].Markdown)
	if err != nil {
		return content.Publication{}, fmt.Errorf("publication %s/%s: %w", topicSlug, folderName, err)
	}
	title, cleanedFirstBody := extractPublicationTitle(firstBody)
	pages[0].Markdown = cleanedFirstBody

	if title == "" {
		title = slug.Humanize(pubSlug)
	}

	rawTags := firstMeta.GetStrings("tags")
	tags := make([]content.Tag, 0, len(rawTags))
	for _, name := range rawTags {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		tags = append(tags, content.Tag{Name: name, Slug: slug.Normalize(name)})
	}

	previewText := preview.Extract(cleanedFirstBody)
	return content.NewPublication(folderName, pubSlug, title, publishedAt, tags, pages, previewText, previewAsset, relatedAssets)
}

func extractPublicationTitle(markdown string) (string, string) {
	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			remaining := append([]string{}, lines[:i]...)
			remaining = append(remaining, lines[i+1:]...)
			body := strings.TrimLeft(strings.Join(remaining, "\n"), "\n")
			return title, body
		}
		break
	}
	return "", markdown
}
