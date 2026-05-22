package sitegen

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"blog-ssg/internal/domain/content"
	"blog-ssg/internal/domain/navigation"
	"blog-ssg/internal/domain/slug"
	"blog-ssg/internal/infra/contentloader"
	"blog-ssg/internal/infra/render"
)

type Generator struct{}

type manifest struct {
	Files         []string          `json:"files"`
	SourceHashes  map[string]string `json:"source_hashes"`
	GeneratorHash string            `json:"generator_hash"`
}

func New() Generator {
	return Generator{}
}

func (Generator) Generate(contentPath, outputPath string) error {
	sourceHashes, err := snapshotSources(contentPath)
	if err != nil {
		return err
	}
	generatorHash, err := snapshotGenerator()
	if err != nil {
		return err
	}

	site, err := contentloader.Load(contentPath)
	if err != nil {
		return err
	}
	oldManifest, err := readManifest(outputPath)
	if err != nil {
		return err
	}

	generatedFiles, err := render.Build(site)
	if err != nil {
		return err
	}

	fullRebuild := oldManifest.SourceHashes == nil || oldManifest.GeneratorHash != generatorHash
	plan := buildWritePlan(site, diffSources(oldManifest.SourceHashes, sourceHashes), fullRebuild)

	var manifestEntries []string
	styleWritten := false

	for _, file := range generatedFiles {
		target := filepath.Join(outputPath, filepath.FromSlash(file.Path))
		if !plan.ShouldWrite(file.Path) && fileExists(target) {
			manifestEntries = append(manifestEntries, filepath.ToSlash(file.Path))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := writeIfChanged(target, file.Content); err != nil {
			return err
		}
		manifestEntries = append(manifestEntries, filepath.ToSlash(file.Path))
	}

	for _, asset := range site.GlobalAssets {
		if asset.PublicPath == "style.css" {
			target := filepath.Join(outputPath, "style.css")
			if !plan.ShouldWrite("style.css") && fileExists(target) {
				manifestEntries = append(manifestEntries, "style.css")
				styleWritten = true
				continue
			}
			body, err := os.ReadFile(asset.SourcePath)
			if err != nil {
				return err
			}
			combined := strings.TrimSpace(string(body)) + "\n\n" + strings.TrimSpace(render.BaseCSS()) + "\n"
			if err := writeIfChanged(target, []byte(combined)); err != nil {
				return err
			}
			manifestEntries = append(manifestEntries, "style.css")
			styleWritten = true
			continue
		}
		target := filepath.Join(outputPath, filepath.FromSlash(asset.PublicPath))
		if !plan.ShouldWrite(asset.PublicPath) && fileExists(target) {
			manifestEntries = append(manifestEntries, filepath.ToSlash(asset.PublicPath))
			continue
		}
		if err := copyFile(asset.SourcePath, target); err != nil {
			return err
		}
		manifestEntries = append(manifestEntries, filepath.ToSlash(asset.PublicPath))
	}

	for _, topic := range site.Topics {
		if topic.Assets.Banner != nil {
			target := filepath.Join(outputPath, filepath.FromSlash(topic.Assets.Banner.PublicPath))
			if !plan.ShouldWrite(topic.Assets.Banner.PublicPath) && fileExists(target) {
				manifestEntries = append(manifestEntries, filepath.ToSlash(topic.Assets.Banner.PublicPath))
			} else {
				if err := copyFile(topic.Assets.Banner.SourcePath, target); err != nil {
					return err
				}
				manifestEntries = append(manifestEntries, filepath.ToSlash(topic.Assets.Banner.PublicPath))
			}
		}
		for _, asset := range topic.Assets.Other {
			target := filepath.Join(outputPath, filepath.FromSlash(asset.PublicPath))
			if !plan.ShouldWrite(asset.PublicPath) && fileExists(target) {
				manifestEntries = append(manifestEntries, filepath.ToSlash(asset.PublicPath))
				continue
			}
			if err := copyFile(asset.SourcePath, target); err != nil {
				return err
			}
			manifestEntries = append(manifestEntries, filepath.ToSlash(asset.PublicPath))
		}
		for _, publication := range topic.Publications {
			for _, asset := range publication.RelatedAssets {
				target := filepath.Join(outputPath, filepath.FromSlash(asset.PublicPath))
				if !plan.ShouldWrite(asset.PublicPath) && fileExists(target) {
					manifestEntries = append(manifestEntries, filepath.ToSlash(asset.PublicPath))
					continue
				}
				if err := copyFile(asset.SourcePath, target); err != nil {
					return err
				}
				manifestEntries = append(manifestEntries, filepath.ToSlash(asset.PublicPath))
			}
		}
	}

	if !styleWritten {
		target := filepath.Join(outputPath, "style.css")
		if plan.ShouldWrite("style.css") || !fileExists(target) {
			if err := writeIfChanged(target, []byte(strings.TrimSpace(render.BaseCSS())+"\n")); err != nil {
				return err
			}
		}
		manifestEntries = append(manifestEntries, "style.css")
	}

	if err := cleanupStale(outputPath, oldManifest.Files, manifestEntries); err != nil {
		return err
	}
	return writeManifest(outputPath, manifest{Files: manifestEntries, SourceHashes: sourceHashes, GeneratorHash: generatorHash})
}

func writeIfChanged(path string, content []byte) error {
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, content) {
		return nil
	}
	return os.WriteFile(path, content, 0o644)
}

func copyFile(sourcePath, targetPath string) error {
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	return writeIfChanged(targetPath, body)
}

func writeManifest(outputPath string, state manifest) error {
	sort.Strings(state.Files)
	manifestPath := filepath.Join(outputPath, ".ssg-manifest.json")
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, body, 0o644)
}

func readManifest(outputPath string) (manifest, error) {
	manifestPath := filepath.Join(outputPath, ".ssg-manifest.json")
	existing, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return manifest{}, nil
		}
		return manifest{}, err
	}

	var old manifest
	if err := json.Unmarshal(existing, &old); err != nil {
		return manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	return old, nil
}

func cleanupStale(outputPath string, oldEntries []string, newEntries []string) error {
	current := map[string]struct{}{}
	for _, entry := range newEntries {
		current[entry] = struct{}{}
	}
	for _, oldPath := range oldEntries {
		if _, ok := current[oldPath]; ok {
			continue
		}
		_ = os.Remove(filepath.Join(outputPath, filepath.FromSlash(oldPath)))
	}
	return nil
}

func snapshotSources(contentPath string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.WalkDir(contentPath, func(currentPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		body, err := os.ReadFile(currentPath)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(contentPath, currentPath)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		result[filepath.ToSlash(rel)] = fmt.Sprintf("%x", sum[:])
		return nil
	})
	return result, err
}

func snapshotGenerator() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	body, err := os.ReadFile(executablePath)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%x", sum[:]), nil
}

func diffSources(oldHashes, newHashes map[string]string) []string {
	seen := map[string]struct{}{}
	var changed []string
	for path, newHash := range newHashes {
		seen[path] = struct{}{}
		if oldHashes == nil || oldHashes[path] != newHash {
			changed = append(changed, path)
		}
	}
	for path := range oldHashes {
		if _, ok := seen[path]; ok {
			continue
		}
		changed = append(changed, path)
	}
	sort.Strings(changed)
	return changed
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type writePlan struct {
	all     bool
	outputs map[string]struct{}
}

func (p writePlan) ShouldWrite(outputPath string) bool {
	if p.all {
		return true
	}
	_, ok := p.outputs[filepath.ToSlash(outputPath)]
	return ok
}

func buildWritePlan(site content.Site, changedSources []string, firstRun bool) writePlan {
	if firstRun {
		return writePlan{all: true}
	}

	plan := writePlan{outputs: map[string]struct{}{}}
	var styleChanged bool
	var rootChanged bool
	var changedTopics = map[string]struct{}{}
	var changedTopicPages = map[string]map[string]struct{}{}
	var changedPublications = map[string]map[string]struct{}{}

	add := func(output string) {
		plan.outputs[filepath.ToSlash(output)] = struct{}{}
	}

	for _, rel := range changedSources {
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) == 1 {
			if rel == "style.css" {
				styleChanged = true
				add("style.css")
				continue
			}
			add(rel)
			continue
		}

		topic := parts[0]
		if parts[1] == "meta" {
			changedTopics[topic] = struct{}{}
			if len(parts) >= 3 && parts[2] == "Config.yaml" {
				rootChanged = true
				for _, siteTopic := range site.Topics {
					if siteTopic.Slug == topic {
						for _, pub := range siteTopic.Publications {
							markPublicationChanged(changedPublications, topic, pub.Slug)
						}
						for _, page := range siteTopic.Pages {
							markTopicPageChanged(changedTopicPages, topic, page.Slug)
						}
						if siteTopic.Assets.Banner != nil {
							add(siteTopic.Assets.Banner.PublicPath)
						}
					}
				}
				continue
			}
			if len(parts) >= 3 && parts[2] == "Links.md" {
				continue
			}
			if len(parts) >= 3 && strings.HasSuffix(strings.ToLower(parts[2]), ".md") {
				pageSlug := strings.TrimSuffix(parts[2], path.Ext(parts[2]))
				markTopicPageChanged(changedTopicPages, topic, slug.Normalize(pageSlug))
				continue
			}
			if len(parts) >= 3 {
				add(path.Join(topic, "assets", "meta", parts[2]))
			}
			continue
		}

		if len(parts) >= 2 {
			changedTopics[topic] = struct{}{}
			folderName := parts[1]
			pubSlug := publicationSlugFromFolder(folderName)
			if pubSlug != "" {
				markPublicationChanged(changedPublications, topic, pubSlug)
			}
			if len(parts) >= 3 && !strings.HasSuffix(strings.ToLower(parts[2]), ".md") && pubSlug != "" {
				add(path.Join(topic, "assets", pubSlug, parts[2]))
			}
		}
	}

	if styleChanged {
		add("style.css")
	}
	if rootChanged {
		add("index.html")
	}
	for topic := range changedTopics {
		add(path.Join(topic, "index.html"))
	}
	for _, topic := range site.Topics {
		if pages, ok := changedTopicPages[topic.Slug]; ok {
			for page := range pages {
				add(path.Join(topic.Slug, page, "index.html"))
			}
		}
		if pubs, ok := changedPublications[topic.Slug]; ok {
			for _, pub := range topic.Publications {
				if _, changed := pubs[pub.Slug]; !changed {
					continue
				}
				for _, page := range pub.Pages {
					add(path.Join(navigation.PublicationPath(topic.Slug, pub.Slug, page.Number), "index.html"))
				}
			}
		}
	}
	return plan
}

func markTopicPageChanged(changed map[string]map[string]struct{}, topicSlug, pageSlug string) {
	if changed[topicSlug] == nil {
		changed[topicSlug] = map[string]struct{}{}
	}
	changed[topicSlug][pageSlug] = struct{}{}
}

func markPublicationChanged(changed map[string]map[string]struct{}, topicSlug, pubSlug string) {
	if changed[topicSlug] == nil {
		changed[topicSlug] = map[string]struct{}{}
	}
	changed[topicSlug][pubSlug] = struct{}{}
}

func publicationSlugFromFolder(folderName string) string {
	if len(folderName) <= len("2006-01-02-") {
		return ""
	}
	if folderName[10] != '-' {
		return ""
	}
	return folderName[11:]
}
