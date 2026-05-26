package render

import (
	"strings"
	"testing"

	"blog-ssg/internal/domain/content"
)

func TestRenderMarkdownWithCustomSyntax(t *testing.T) {
	html, err := renderMarkdown(`## Заголовок

Текст со ссылкой [[meta/where-to-play.md|Играть]] и картинкой ![[preview.png]]

[font="Riichi-Mahjong-Colorful"][rot-90]5p*[/rot-90]🀛🀜[/font]`, markdownContext{
		TopicSlug:       "riichirocks",
		PublicationSlug: "kuikae",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, html, "<h2")
	assertContains(t, html, `href="where-to-play/"`)
	assertContains(t, html, `src="../assets/kuikae/preview.png"`)
	assertContains(t, html, `class="font-riichi-mahjong-colorful"`)
	assertContains(t, html, `class="rot-90-char"`)
}

func TestRenderMarkdownRejectsUnclosedFontTag(t *testing.T) {
	_, err := renderMarkdown(`[font="Riichi-Mahjong-Colorful"]🀚🀛`, markdownContext{})
	if err == nil {
		t.Fatalf("expected error for unclosed font tag")
	}
}

func TestRenderMarkdownRejectsMismatchedClosingTag(t *testing.T) {
	_, err := renderMarkdown(`[font="A"]x[/font-b]`, markdownContext{})
	if err == nil {
		t.Fatalf("expected error for mismatched closing tag")
	}
}

func TestRenderMarkdownRejectsBrokenWikiImage(t *testing.T) {
	_, err := renderMarkdown(`![[ ]]`, markdownContext{})
	if err == nil {
		t.Fatalf("expected error for empty wiki image target")
	}
}

func TestRenderMarkdownHighlightsFencedCodeBlock(t *testing.T) {
	html, err := renderMarkdown("```Python\ndef f():\n    return 1\n```", markdownContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, html, `class="chroma-chroma"`)
	assertContains(t, html, `class="chroma-k"`)
}

func TestRenderMarkdownExpandsMahjongShorthandInsideFontTag(t *testing.T) {
	html, err := renderMarkdown(`[font="Riichi-Mahjong-Colorful"]667s[/font]`, markdownContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, html, `6s6s7s`)
}

func TestRenderMarkdownExpandsMahjongShorthandAcrossSuits(t *testing.T) {
	html, err := renderMarkdown(`[font="Riichi-Mahjong-Colorful"]667s34668m2357p2z[/font]`, markdownContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, html, `6s6s7s3m4m6m6m8m2p3p5p7p2z`)
}

func TestRenderNotFoundPageLinksToSingleTopicHome(t *testing.T) {
	html, err := renderNotFoundPage(content.Site{
		Topics: []content.Topic{{Slug: "programming"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, string(html), `href="programming/"`)
	assertContains(t, string(html), `"programming"`)
}

func TestRenderNotFoundPageCanResolveCurrentTopic(t *testing.T) {
	html, err := renderNotFoundPage(content.Site{
		Topics: []content.Topic{
			{Slug: "programming"},
			{Slug: "riichi-mahjong"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, string(html), `href="./"`)
	assertContains(t, string(html), `"programming"`)
	assertContains(t, string(html), `"riichi-mahjong"`)
	assertContains(t, string(html), `link.href = '/' + (prefix ? prefix + '/' : '') + topic + '/';`)
}

func assertContains(t *testing.T, actual string, expected string) {
	t.Helper()
	if !strings.Contains(actual, expected) {
		t.Fatalf("expected substring %q in %q", expected, actual)
	}
}
