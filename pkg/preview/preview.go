package preview

import "strings"

func Extract(markdown string) string {
	blocks := strings.Split(markdown, "\n\n")
	for _, block := range blocks {
		text := strings.TrimSpace(block)
		if text == "" {
			continue
		}
		if strings.HasPrefix(text, "#") {
			continue
		}
		if strings.HasPrefix(text, "![") || strings.HasPrefix(text, "![[") {
			continue
		}
		text = strings.ReplaceAll(text, "\n", " ")
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		return text
	}
	return ""
}
