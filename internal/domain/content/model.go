package content

import "time"

type Site struct {
	Topics       []Topic
	GlobalAssets []Asset
}

type Topic struct {
	Slug         string
	Config       TopicConfig
	Links        []TopicLink
	Pages        []TopicPage
	Publications []Publication
	Assets       TopicAssets
}

type TopicConfig struct {
	TopicTitle  string
	TopicHeader string
	Theme       ThemeTokens
}

type ThemeTokens struct {
	Background  string
	Text        string
	Muted       string
	Accent      string
	Surface     string
	InverseText string
	Overlay     string
}

type TopicLink struct {
	Label  string
	Target string
	Kind   LinkKind
}

type LinkKind string

const (
	LinkKindExternal LinkKind = "external"
	LinkKindInternal LinkKind = "internal"
)

type TopicPage struct {
	Slug     string
	Title    string
	Markdown string
}

type Publication struct {
	FolderName    string
	Slug          string
	Title         string
	PublishedAt   time.Time
	Tags          []Tag
	Pages         []PublicationPage
	PreviewText   string
	PreviewAsset  *Asset
	RelatedAssets []Asset
}

type PublicationPage struct {
	Number   int
	Markdown string
}

type Tag struct {
	Name string
	Slug string
}

type TopicAssets struct {
	Banner *Asset
	Other  []Asset
}

type Asset struct {
	SourcePath string
	PublicPath string
	Kind       AssetKind
}

type AssetKind string

const (
	AssetKindGeneric    AssetKind = "generic"
	AssetKindPreview    AssetKind = "preview"
	AssetKindBanner     AssetKind = "banner"
	AssetKindStylesheet AssetKind = "stylesheet"
	AssetKindFont       AssetKind = "font"
)
