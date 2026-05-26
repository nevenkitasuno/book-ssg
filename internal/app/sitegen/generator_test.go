package sitegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	outputDir := t.TempDir()
	gen := New()

	if err := gen.Generate("../../../examples/content", outputDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFiles := []string{
		"index.html",
		"404.html",
		"_redirects",
		"style.css",
		filepath.Join("riichi-mahjong", "index.html"),
		filepath.Join("riichi-mahjong", "mahjong-post", "index.html"),
		filepath.Join("fonts", "Riichi-Mahjong-Colorful.woff2"),
	}
	for _, rel := range expectedFiles {
		if _, err := os.Stat(filepath.Join(outputDir, rel)); err != nil {
			t.Fatalf("expected file %s: %v", rel, err)
		}
	}

	redirectsBody, err := os.ReadFile(filepath.Join(outputDir, "_redirects"))
	if err != nil {
		t.Fatalf("read redirects fallback: %v", err)
	}
	if string(redirectsBody) != "/* /404.html 404\n" {
		t.Fatalf("unexpected redirects fallback: %q", string(redirectsBody))
	}

	publicationBody, err := os.ReadFile(filepath.Join(outputDir, "riichi-mahjong", "mahjong-post", "index.html"))
	if err != nil {
		t.Fatalf("read publication page: %v", err)
	}
	if !strings.Contains(string(publicationBody), "Пост про маджонг") {
		t.Fatalf("publication page should contain title")
	}
}

func TestGenerateIncrementalPublicationChange(t *testing.T) {
	contentDir := t.TempDir()
	copyFixtureDir(t, "../../../examples/content", contentDir)

	outputDir := t.TempDir()
	gen := New()
	if err := gen.Generate(contentDir, outputDir); err != nil {
		t.Fatalf("initial generate: %v", err)
	}

	rootPath := filepath.Join(outputDir, "index.html")
	topicPath := filepath.Join(outputDir, "riichi-mahjong", "index.html")
	pubPath := filepath.Join(outputDir, "riichi-mahjong", "mahjong-post", "index.html")

	rootBefore := mustModTime(t, rootPath)
	topicBefore := mustModTime(t, topicPath)
	pubBefore := mustModTime(t, pubPath)

	time.Sleep(1100 * time.Millisecond)

	sourcePath := filepath.Join(contentDir, "riichi-mahjong", "2026-05-03-mahjong-post", "1.md")
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	updated := strings.Replace(string(body), "Пример использования маджонг шрифта и изображений", "Обновлённый текстовый превью-абзац про маджонг", 1)
	if err := os.WriteFile(sourcePath, []byte(updated), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if err := gen.Generate(contentDir, outputDir); err != nil {
		t.Fatalf("incremental generate: %v", err)
	}

	rootAfter := mustModTime(t, rootPath)
	topicAfter := mustModTime(t, topicPath)
	pubAfter := mustModTime(t, pubPath)

	if !rootAfter.Equal(rootBefore) {
		t.Fatalf("root index should not be rebuilt for publication content change")
	}
	if !topicAfter.After(topicBefore) {
		t.Fatalf("topic index should be rebuilt when preview changes")
	}
	if !pubAfter.After(pubBefore) {
		t.Fatalf("publication page should be rebuilt when source changes")
	}
}

func TestGenerateNoChangesDoesNotRewriteOutputs(t *testing.T) {
	contentDir := t.TempDir()
	copyFixtureDir(t, "../../../examples/content", contentDir)

	outputDir := t.TempDir()
	gen := New()
	if err := gen.Generate(contentDir, outputDir); err != nil {
		t.Fatalf("initial generate: %v", err)
	}

	rootPath := filepath.Join(outputDir, "index.html")
	topicPath := filepath.Join(outputDir, "riichi-mahjong", "index.html")
	pubPath := filepath.Join(outputDir, "riichi-mahjong", "mahjong-post", "index.html")
	stylePath := filepath.Join(outputDir, "style.css")

	rootBefore := mustModTime(t, rootPath)
	topicBefore := mustModTime(t, topicPath)
	pubBefore := mustModTime(t, pubPath)
	styleBefore := mustModTime(t, stylePath)

	time.Sleep(1100 * time.Millisecond)

	if err := gen.Generate(contentDir, outputDir); err != nil {
		t.Fatalf("second generate: %v", err)
	}

	if !mustModTime(t, rootPath).Equal(rootBefore) {
		t.Fatalf("root index should not be rewritten when nothing changed")
	}
	if !mustModTime(t, topicPath).Equal(topicBefore) {
		t.Fatalf("topic index should not be rewritten when nothing changed")
	}
	if !mustModTime(t, pubPath).Equal(pubBefore) {
		t.Fatalf("publication page should not be rewritten when nothing changed")
	}
	if !mustModTime(t, stylePath).Equal(styleBefore) {
		t.Fatalf("style should not be rewritten when nothing changed")
	}
}

func TestGenerateRemovesDeletedPublicationOutputs(t *testing.T) {
	contentDir := t.TempDir()
	copyFixtureDir(t, "../../../examples/content", contentDir)

	outputDir := t.TempDir()
	gen := New()
	if err := gen.Generate(contentDir, outputDir); err != nil {
		t.Fatalf("initial generate: %v", err)
	}

	publicationDir := filepath.Join(contentDir, "riichi-mahjong", "2026-05-03-mahjong-post")
	if err := os.RemoveAll(publicationDir); err != nil {
		t.Fatalf("remove publication source: %v", err)
	}

	if err := gen.Generate(contentDir, outputDir); err != nil {
		t.Fatalf("generate after deletion: %v", err)
	}

	removedOutputs := []string{
		filepath.Join(outputDir, "riichi-mahjong", "mahjong-post", "index.html"),
		filepath.Join(outputDir, "riichi-mahjong", "assets", "mahjong-post", "image-by-Tairagi-Makoto.png"),
	}
	for _, output := range removedOutputs {
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("expected stale output to be removed: %s", output)
		}
	}
}

func copyFixtureDir(t *testing.T, sourceRoot, targetRoot string) {
	t.Helper()
	err := filepath.Walk(sourceRoot, func(currentPath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourceRoot, currentPath)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetRoot, rel)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		body, err := os.ReadFile(currentPath)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, body, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture dir: %v", err)
	}
}

func mustModTime(t *testing.T, filePath string) time.Time {
	t.Helper()
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat %s: %v", filePath, err)
	}
	return info.ModTime()
}
