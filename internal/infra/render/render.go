package render

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"blog-ssg/internal/domain/content"
	"blog-ssg/internal/domain/navigation"
	"blog-ssg/internal/domain/slug"
	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	gomarkdown "github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	markdownhtml "github.com/gomarkdown/markdown/html"
	markdownparser "github.com/gomarkdown/markdown/parser"
)

type GeneratedFile struct {
	Path    string
	Content []byte
}

type publicationPageView struct {
	SiteTitle        string
	TopicTitle       string
	SiteRootPath     string
	TopicHomePath    string
	FaviconPath      string
	PublicationTitle string
	Tags             []tagQueryView
	BodyHTML         template.HTML
	ThemeCSS         template.CSS
	NextPath         string
	StartPath        string
	ShowNext         bool
	ShowStart        bool
}

type cardView struct {
	Title       string
	Href        string
	PreviewText string
	PreviewURL  string
	PublishedAt string
	Tags        []content.Tag
	TagQuery    []tagQueryView
	DataTags    string
}

type tagQueryView struct {
	Name string
	Href string
}

type yearSectionView struct {
	ID    string
	Year  string
	Cards []cardView
}

type topicLinkView struct {
	Label string
	Href  string
}

type topicPageView struct {
	SiteTitle   string
	TopicTitle  string
	TopicHeader string
	ThemeCSS    template.CSS
	FaviconPath string
	BannerURL   string
	Links       []topicLinkView
	Years       []yearSectionView
}

type siteIndexView struct {
	SiteTitle   string
	FaviconPath string
	Topics      []siteTopicView
}

type siteTopicView struct {
	Title string
	Href  string
}

type simplePageView struct {
	SiteTitle     string
	TopicTitle    string
	Title         string
	SiteRootPath  string
	TopicHomePath string
	ThemeCSS      template.CSS
	FaviconPath   string
	BodyHTML      template.HTML
}

type notFoundPageView struct {
	FaviconPath string
}

//go:embed templates/*
var embeddedTemplates embed.FS

var (
	syntaxHTMLFormatter = chromahtml.New(
		chromahtml.WithClasses(true),
		chromahtml.ClassPrefix("chroma-"),
		chromahtml.TabWidth(2),
	)
	syntaxHighlightStyle = styles.Get("github-dark")
)

func Build(site content.Site) ([]GeneratedFile, error) {
	var files []GeneratedFile

	indexBytes, err := renderSiteIndex(site)
	if err != nil {
		return nil, err
	}
	files = append(files, GeneratedFile{Path: "index.html", Content: indexBytes})
	notFoundBytes, err := renderNotFoundPage(site)
	if err != nil {
		return nil, err
	}
	files = append(files, GeneratedFile{Path: "404.html", Content: notFoundBytes})

	for _, topic := range site.Topics {
		topicBytes, err := renderTopicIndex(topic, site)
		if err != nil {
			return nil, err
		}
		files = append(files, GeneratedFile{Path: path.Join(topic.Slug, "index.html"), Content: topicBytes})

		for _, page := range topic.Pages {
			pageBytes, err := renderTopicPage(topic, site, page)
			if err != nil {
				return nil, err
			}
			files = append(files, GeneratedFile{
				Path:    path.Join(topic.Slug, page.Slug, "index.html"),
				Content: pageBytes,
			})
		}

		for _, publication := range topic.Publications {
			for _, pubPage := range publication.Pages {
				pageBytes, err := renderPublicationPage(topic, site, publication, pubPage)
				if err != nil {
					return nil, err
				}
				pubPath := navigation.PublicationPath(topic.Slug, publication.Slug, pubPage.Number)
				files = append(files, GeneratedFile{
					Path:    path.Join(pubPath, "index.html"),
					Content: pageBytes,
				})
			}
		}
	}

	return files, nil
}

func renderPublicationPage(topic content.Topic, site content.Site, publication content.Publication, page content.PublicationPage) ([]byte, error) {
	body, err := renderMarkdown(page.Markdown, markdownContext{
		TopicSlug:       topic.Slug,
		PublicationSlug: publication.Slug,
		TopicPages:      topic.Pages,
	})
	if err != nil {
		return nil, err
	}

	nav := navigation.Build(topic.Slug, publication.Slug, page.Number, len(publication.Pages))
	view := publicationPageView{
		SiteTitle:        siteTitle(site),
		TopicTitle:       topic.Config.TopicHeader,
		SiteRootPath:     relativeToRoot(nav.Current),
		TopicHomePath:    relativeFrom(path.Join(nav.Current, "index.html"), topic.Slug+"/"),
		FaviconPath:      relativeFrom(path.Join(nav.Current, "index.html"), topicFaviconPath(topic, site)),
		PublicationTitle: publication.Title,
		BodyHTML:         template.HTML(body),
		ThemeCSS:         themeCSS(topic.Config.Theme),
		NextPath:         relativeFrom(nav.Current, nav.Next),
		StartPath:        relativeFrom(nav.Current, nav.Start),
		ShowNext:         nav.Next != "",
		ShowStart:        nav.IsLast && len(publication.Pages) > 1,
	}
	for _, tag := range publication.Tags {
		view.Tags = append(view.Tags, tagQueryView{
			Name: tag.Name,
			Href: view.TopicHomePath + "?tag=" + url.QueryEscape(tag.Slug),
		})
	}

	return executeTemplate("templates/publication.html.tmpl", view)
}

func renderTopicIndex(topic content.Topic, site content.Site) ([]byte, error) {
	sections := map[string][]cardView{}
	var years []string
	for _, pub := range topic.Publications {
		year := pub.PublishedAt.Format("2006")
		if _, exists := sections[year]; !exists {
			years = append(years, year)
		}
		card := cardView{
			Title:       pub.Title,
			Href:        pub.Slug + "/",
			PreviewText: pub.PreviewText,
			PublishedAt: pub.PublishedAt.Format("2006-01-02"),
			DataTags:    strings.Join(tagSlugs(pub.Tags), " "),
		}
		if pub.PreviewAsset != nil {
			card.PreviewURL = relativeFrom(path.Join(topic.Slug, "index.html"), pub.PreviewAsset.PublicPath)
		}
		for _, tag := range pub.Tags {
			card.Tags = append(card.Tags, tag)
			card.TagQuery = append(card.TagQuery, tagQueryView{
				Name: tag.Name,
				Href: "?tag=" + url.QueryEscape(tag.Slug),
			})
		}
		sections[year] = append(sections[year], card)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(years)))

	var yearSections []yearSectionView
	for _, year := range years {
		yearSections = append(yearSections, yearSectionView{
			ID:    "year-" + year,
			Year:  year,
			Cards: sections[year],
		})
	}

	var links []topicLinkView
	for _, link := range topic.Links {
		href := link.Target
		if link.Kind == content.LinkKindInternal {
			href = resolveTopicLink(topic, link.Target)
		}
		links = append(links, topicLinkView{Label: link.Label, Href: href})
	}

	bannerURL := ""
	if topic.Assets.Banner != nil {
		bannerURL = relativeFrom(path.Join(topic.Slug, "index.html"), topic.Assets.Banner.PublicPath)
	}

	view := topicPageView{
		SiteTitle:   siteTitle(site),
		TopicTitle:  topic.Config.TopicTitle,
		TopicHeader: topic.Config.TopicHeader,
		ThemeCSS:    themeCSS(topic.Config.Theme),
		FaviconPath: relativeFrom(path.Join(topic.Slug, "index.html"), topicFaviconPath(topic, site)),
		BannerURL:   bannerURL,
		Links:       links,
		Years:       yearSections,
	}

	return executeTemplate("templates/topic-index.html.tmpl", view)
}

func renderTopicPage(topic content.Topic, site content.Site, page content.TopicPage) ([]byte, error) {
	body, err := renderMarkdown(page.Markdown, markdownContext{
		TopicSlug:  topic.Slug,
		TopicPages: topic.Pages,
	})
	if err != nil {
		return nil, err
	}

	view := simplePageView{
		SiteTitle:     siteTitle(site),
		TopicTitle:    topic.Config.TopicHeader,
		Title:         page.Title,
		SiteRootPath:  relativeToRoot(path.Join(topic.Slug, page.Slug, "index.html")),
		TopicHomePath: relativeFrom(path.Join(topic.Slug, page.Slug, "index.html"), topic.Slug+"/"),
		ThemeCSS:      themeCSS(topic.Config.Theme),
		FaviconPath:   relativeFrom(path.Join(topic.Slug, page.Slug, "index.html"), topicFaviconPath(topic, site)),
		BodyHTML:      template.HTML(body),
	}
	return executeTemplate("templates/topic-page.html.tmpl", view)
}

func renderSiteIndex(site content.Site) ([]byte, error) {
	view := siteIndexView{SiteTitle: siteTitle(site), FaviconPath: faviconPath(site)}
	for _, topic := range site.Topics {
		view.Topics = append(view.Topics, siteTopicView{
			Title: topic.Config.TopicTitle,
			Href:  topic.Slug + "/",
		})
	}
	return executeTemplate("templates/site-index.html.tmpl", view)
}

func renderNotFoundPage(site content.Site) ([]byte, error) {
	return executeTemplate("templates/404.html.tmpl", notFoundPageView{
		FaviconPath: faviconPath(site),
	})
}

func siteTitle(site content.Site) string {
	if len(site.Topics) == 1 {
		return site.Topics[0].Config.TopicTitle
	}
	return "Topics"
}

func executeTemplate(templatePath string, view any) ([]byte, error) {
	source, err := embeddedTemplates.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}
	tpl, err := template.New(path.Base(templatePath)).Parse(string(source))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, view); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func tagSlugs(tags []content.Tag) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		result = append(result, tag.Slug)
	}
	return result
}

func faviconPath(site content.Site) string {
	for _, asset := range site.GlobalAssets {
		if strings.EqualFold(path.Base(asset.PublicPath), "favicon.ico") {
			return asset.PublicPath
		}
	}
	return ""
}

func topicFaviconPath(topic content.Topic, site content.Site) string {
	for _, asset := range topic.Assets.Other {
		if strings.EqualFold(path.Base(asset.PublicPath), "favicon.ico") {
			return asset.PublicPath
		}
	}
	return faviconPath(site)
}

func relativeFrom(currentPagePath, targetPath string) string {
	if targetPath == "" {
		return ""
	}
	currentDir := currentDirectory(currentPagePath)
	rel, err := filepath.Rel(filepath.FromSlash(currentDir), filepath.FromSlash(targetPath))
	if err != nil {
		return targetPath
	}
	return filepath.ToSlash(rel)
}

func relativeToRoot(currentPagePath string) string {
	currentDir := currentDirectory(currentPagePath)
	if currentDir == "" {
		return "./"
	}
	depth := len(strings.Split(currentDir, "/"))
	return strings.Repeat("../", depth)
}

func currentDirectory(currentPagePath string) string {
	if strings.HasSuffix(currentPagePath, "/") {
		return strings.TrimSuffix(currentPagePath, "/")
	}
	dir := path.Dir(currentPagePath)
	if dir == "." {
		return ""
	}
	return dir
}

func resolveTopicLink(topic content.Topic, target string) string {
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "meta/") && strings.HasSuffix(strings.ToLower(target), ".md") {
		name := strings.TrimSuffix(path.Base(target), path.Ext(target))
		return slug.Normalize(name) + "/"
	}
	return target
}

func themeCSS(tokens content.ThemeTokens) template.CSS {
	return template.CSS(fmt.Sprintf(`:root{
--color-background:%s;
--color-text:%s;
--color-muted:%s;
--color-accent:%s;
--color-surface:%s;
--color-inverse-text:%s;
--color-overlay:%s;
--color-heading:var(--color-text);
--color-border:color-mix(in srgb, var(--color-muted) 28%%, transparent);
--color-border-soft:color-mix(in srgb, var(--color-muted) 18%%, transparent);
--color-code-bg:color-mix(in srgb, var(--color-surface) 88%%, var(--color-background));
--color-code-border:color-mix(in srgb, var(--color-muted) 24%%, transparent);
}
%s`, tokens.Background, tokens.Text, tokens.Muted, tokens.Accent, tokens.Surface, tokens.InverseText, tokens.Overlay, syntaxHighlightCSS()))
}

type markdownContext struct {
	TopicSlug       string
	PublicationSlug string
	TopicPages      []content.TopicPage
}

func renderMarkdown(input string, ctx markdownContext) (string, error) {
	preprocessed, fragments, err := preprocessCustomSyntax(input, ctx)
	if err != nil {
		return "", err
	}
	extensions := markdownparser.CommonExtensions | markdownparser.AutoHeadingIDs
	parser := markdownparser.NewWithExtensions(extensions)
	renderer := markdownhtml.NewRenderer(markdownhtml.RendererOptions{
		Flags:          markdownhtml.CommonFlags,
		RenderNodeHook: renderNodeHook,
	})
	rendered := string(gomarkdown.ToHTML([]byte(preprocessed), parser, renderer))
	for i := len(fragments) - 1; i >= 0; i-- {
		fragment := fragments[i]
		rendered = strings.ReplaceAll(rendered, placeholderToken(i), fragment)
	}
	return rendered, nil
}

func renderNodeHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	codeBlock, ok := node.(*ast.CodeBlock)
	if !ok {
		return ast.GoToNext, false
	}
	if !entering {
		return ast.GoToNext, true
	}
	if err := renderHighlightedCodeBlock(w, codeBlock); err != nil {
		writeFallbackCodeBlock(w, codeBlock)
	}
	return ast.GoToNext, true
}

func renderHighlightedCodeBlock(w io.Writer, codeBlock *ast.CodeBlock) error {
	source := string(codeBlock.Literal)
	lang := ""
	if infoFields := strings.Fields(string(codeBlock.Info)); len(infoFields) > 0 {
		lang = strings.ToLower(strings.TrimSpace(infoFields[0]))
	}
	if lang == "" {
		lang = detectCodeLanguage(source)
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Analyse(source)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return err
	}
	return syntaxHTMLFormatter.Format(w, syntaxHighlightStyle, iterator)
}

func writeFallbackCodeBlock(w io.Writer, codeBlock *ast.CodeBlock) {
	_, _ = io.WriteString(w, `<pre><code>`)
	_, _ = io.WriteString(w, html.EscapeString(string(codeBlock.Literal)))
	_, _ = io.WriteString(w, `</code></pre>`)
}

func detectCodeLanguage(source string) string {
	if lexer := lexers.Analyse(source); lexer != nil {
		if cfg := lexer.Config(); cfg != nil && len(cfg.Aliases) > 0 {
			return cfg.Aliases[0]
		}
	}
	return ""
}

func syntaxHighlightCSS() string {
	if syntaxHighlightStyle == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := syntaxHTMLFormatter.WriteCSS(&buf, syntaxHighlightStyle); err != nil {
		return ""
	}
	return buf.String()
}

func preprocessCustomSyntax(input string, ctx markdownContext) (string, []string, error) {
	parser := inlineParser{
		input: input,
		ctx:   ctx,
	}
	out, err := parser.parse("")
	if err != nil {
		return "", nil, err
	}
	return out, parser.fragments, nil
}

type inlineParser struct {
	input     string
	pos       int
	ctx       markdownContext
	fragments []string
}

func (p *inlineParser) parse(until string) (string, error) {
	return p.parseWithFont(until, "")
}

func (p *inlineParser) parseWithFont(until string, fontClass string) (string, error) {
	var out strings.Builder
	for p.pos < len(p.input) {
		if until != "" && strings.HasPrefix(p.input[p.pos:], until) {
			p.pos += len(until)
			return out.String(), nil
		}

		switch {
		case strings.HasPrefix(p.input[p.pos:], "[/"):
			return "", fmt.Errorf("unexpected closing tag near %q", previewFragment(p.input[p.pos:]))
		case strings.HasPrefix(p.input[p.pos:], "[rot-90]"):
			p.pos += len("[rot-90]")
			inner, err := p.parseWithFont("[/rot-90]", fontClass)
			if err != nil {
				return "", err
			}
			out.WriteString(p.addFragment(`<span class="rot-90"><span class="rot-90-char">` + inner + `</span></span>`))
		case strings.HasPrefix(p.input[p.pos:], "[font="):
			fontFamily, err := p.parseFontAttributeOpen()
			if err != nil {
				return "", err
			}
			className := slug.Normalize(fontFamily)
			if className == "" {
				return "", fmt.Errorf("font family %q produces empty class", fontFamily)
			}
			inner, err := p.parseWithFont("[/font]", className)
			if err != nil {
				return "", err
			}
			classEscaped := html.EscapeString(className)
			out.WriteString(p.addFragment(`<span class="font-` + classEscaped + `"><span class="font-` + classEscaped + `-run">` + inner + `</span></span>`))
		case strings.HasPrefix(p.input[p.pos:], "[font-"):
			className, err := p.parseFontOpen()
			if err != nil {
				return "", err
			}
			inner, err := p.parseWithFont("[/font-"+className+"]", className)
			if err != nil {
				return "", err
			}
			classEscaped := html.EscapeString(className)
			out.WriteString(p.addFragment(`<span class="font-` + classEscaped + `"><span class="font-` + classEscaped + `-run">` + inner + `</span></span>`))
		case strings.HasPrefix(p.input[p.pos:], "![["):
			imageHTML, err := p.parseWikiImage()
			if err != nil {
				return "", err
			}
			out.WriteString(p.addFragment(imageHTML))
		case strings.HasPrefix(p.input[p.pos:], "[["):
			linkHTML, err := p.parseWikiLink()
			if err != nil {
				return "", err
			}
			out.WriteString(p.addFragment(linkHTML))
		default:
			text := p.readTextSegment(until)
			if usesMahjongShorthand(fontClass) {
				text = expandMahjongShorthand(text)
			}
			out.WriteString(text)
		}
	}
	if until != "" {
		return "", fmt.Errorf("missing closing tag %q", until)
	}
	return out.String(), nil
}

func (p *inlineParser) readTextSegment(until string) string {
	start := p.pos
	for p.pos < len(p.input) {
		if until != "" && strings.HasPrefix(p.input[p.pos:], until) {
			break
		}
		if strings.HasPrefix(p.input[p.pos:], "[/") ||
			strings.HasPrefix(p.input[p.pos:], "[rot-90]") ||
			strings.HasPrefix(p.input[p.pos:], "[font=") ||
			strings.HasPrefix(p.input[p.pos:], "[font-") ||
			strings.HasPrefix(p.input[p.pos:], "![[") ||
			strings.HasPrefix(p.input[p.pos:], "[[") {
			break
		}
		p.pos++
	}
	return p.input[start:p.pos]
}

func usesMahjongShorthand(fontClass string) bool {
	switch fontClass {
	case "riichi-mahjong-colorful", "mahjong-colored":
		return true
	default:
		return false
	}
}

func expandMahjongShorthand(input string) string {
	var out strings.Builder
	for i := 0; i < len(input); {
		if !isMahjongDigit(input[i]) {
			out.WriteByte(input[i])
			i++
			continue
		}

		start := i
		for i < len(input) && isMahjongDigit(input[i]) {
			i++
		}
		if i >= len(input) || !isMahjongSuit(input[i]) {
			out.WriteString(input[start:i])
			continue
		}

		digits := input[start:i]
		suit := input[i]
		i++
		for j := 0; j < len(digits); j++ {
			out.WriteByte(digits[j])
			out.WriteByte(suit)
		}
	}
	return out.String()
}

func isMahjongDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isMahjongSuit(b byte) bool {
	switch b {
	case 'm', 'p', 's', 'z', 'M', 'P', 'S', 'Z':
		return true
	default:
		return false
	}
}

func (p *inlineParser) parseFontOpen() (string, error) {
	start := p.pos + len("[font-")
	end := strings.IndexByte(p.input[start:], ']')
	if end < 0 {
		return "", fmt.Errorf("unterminated font tag near %q", previewFragment(p.input[p.pos:]))
	}
	end += start
	className := p.input[start:end]
	if className == "" {
		return "", fmt.Errorf("font tag class is empty near %q", previewFragment(p.input[p.pos:]))
	}
	for _, r := range className {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_') {
			return "", fmt.Errorf("invalid font tag class %q", className)
		}
	}
	p.pos = end + 1
	return className, nil
}

func (p *inlineParser) parseFontAttributeOpen() (string, error) {
	start := p.pos + len("[font=")
	if start >= len(p.input) {
		return "", fmt.Errorf("unterminated font tag near %q", previewFragment(p.input[p.pos:]))
	}
	quote := p.input[start]
	if quote != '"' && quote != '\'' {
		return "", fmt.Errorf("font tag value must be quoted near %q", previewFragment(p.input[p.pos:]))
	}
	valueStart := start + 1
	valueEnd := strings.IndexByte(p.input[valueStart:], quote)
	if valueEnd < 0 {
		return "", fmt.Errorf("unterminated font tag near %q", previewFragment(p.input[p.pos:]))
	}
	valueEnd += valueStart
	if valueEnd+1 >= len(p.input) || p.input[valueEnd+1] != ']' {
		return "", fmt.Errorf("invalid font tag syntax near %q", previewFragment(p.input[p.pos:]))
	}
	fontFamily := strings.TrimSpace(p.input[valueStart:valueEnd])
	if fontFamily == "" {
		return "", fmt.Errorf("font family is empty near %q", previewFragment(p.input[p.pos:]))
	}
	p.pos = valueEnd + 2
	return fontFamily, nil
}

func (p *inlineParser) parseWikiImage() (string, error) {
	start := p.pos + len("![[")
	end := strings.Index(p.input[start:], "]]")
	if end < 0 {
		return "", fmt.Errorf("unterminated wiki image near %q", previewFragment(p.input[p.pos:]))
	}
	end += start
	target := strings.TrimSpace(p.input[start:end])
	if target == "" {
		return "", fmt.Errorf("wiki image target is empty")
	}
	if strings.Contains(target, "[") || strings.Contains(target, "]") {
		return "", fmt.Errorf("invalid wiki image target %q", target)
	}
	p.pos = end + 2
	src := path.Join("..", "assets", p.ctx.PublicationSlug, target)
	if p.ctx.PublicationSlug == "" {
		src = target
	}
	return `<img src="` + html.EscapeString(src) + `" alt="">`, nil
}

func (p *inlineParser) parseWikiLink() (string, error) {
	start := p.pos + len("[[")
	end := strings.Index(p.input[start:], "]]")
	if end < 0 {
		return "", fmt.Errorf("unterminated wiki link near %q", previewFragment(p.input[p.pos:]))
	}
	end += start
	inner := p.input[start:end]
	parts := strings.SplitN(inner, "|", 2)
	target := strings.TrimSpace(parts[0])
	if target == "" {
		return "", fmt.Errorf("wiki link target is empty")
	}
	label := target
	if len(parts) == 2 {
		label = strings.TrimSpace(parts[1])
		if label == "" {
			return "", fmt.Errorf("wiki link label is empty for target %q", target)
		}
	}
	p.pos = end + 2
	href := target
	if strings.HasPrefix(target, "meta/") && strings.HasSuffix(strings.ToLower(target), ".md") {
		href = slug.Normalize(strings.TrimSuffix(path.Base(target), path.Ext(target))) + "/"
	}
	return `<a href="` + html.EscapeString(href) + `">` + html.EscapeString(label) + `</a>`, nil
}

func (p *inlineParser) addFragment(fragment string) string {
	token := placeholderToken(len(p.fragments))
	p.fragments = append(p.fragments, fragment)
	return token
}

func placeholderToken(index int) string {
	return fmt.Sprintf("SSGHTMLPLACEHOLDER%dTOKEN", index)
}

func previewFragment(input string) string {
	input = strings.TrimSpace(input)
	if len(input) > 24 {
		return input[:24] + "..."
	}
	return input
}

func BaseCSS() string {
	body, err := embeddedTemplates.ReadFile("templates/base.css")
	if err != nil {
		panic(err)
	}
	return string(body)
}
