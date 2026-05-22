package slug

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	gosimpleslug "github.com/gosimple/slug"
)

func ParsePublicationDir(name string) (time.Time, string, error) {
	base := filepath.Base(name)
	if len(base) < len("2006-01-02-a") {
		return time.Time{}, "", fmt.Errorf("invalid publication directory: %s", name)
	}
	datePart := base[:10]
	if len(base) < 12 || base[10] != '-' {
		return time.Time{}, "", fmt.Errorf("invalid publication directory: %s", name)
	}
	parsedDate, err := time.Parse("2006-01-02", datePart)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid publication date %q: %w", datePart, err)
	}
	rawSlug := strings.TrimSpace(base[11:])
	if rawSlug == "" {
		return time.Time{}, "", fmt.Errorf("invalid publication directory: missing slug in %s", name)
	}
	return parsedDate, Normalize(rawSlug), nil
}

func Normalize(input string) string {
	normalized := gosimpleslug.Make(strings.TrimSpace(input))
	if normalized == "" {
		return ""
	}
	return normalized
}

func Humanize(input string) string {
	parts := strings.Split(strings.TrimSpace(input), "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}
