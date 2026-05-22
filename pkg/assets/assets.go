package assets

import (
	"path/filepath"
	"strings"

	"blog-ssg/internal/domain/content"
)

func Classify(path string) content.AssetKind {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case base == "preview.png" || base == "preview.jpg" || base == "preview.jpeg" || base == "preview.webp":
		return content.AssetKindPreview
	case base == "top_banner.png" || base == "top_banner.jpg" || base == "top_banner.jpeg" || base == "top_banner.webp":
		return content.AssetKindBanner
	case ext == ".css":
		return content.AssetKindStylesheet
	case ext == ".woff2" || ext == ".woff" || ext == ".ttf" || ext == ".otf":
		return content.AssetKindFont
	default:
		return content.AssetKindGeneric
	}
}
