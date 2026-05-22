package frontmatter

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Metadata struct {
	scalars map[string]string
	lists   map[string][]string
}

func (m Metadata) GetString(key string) string {
	return m.scalars[key]
}

func (m Metadata) GetStrings(key string) []string {
	return append([]string(nil), m.lists[key]...)
}

func Parse(document string) (Metadata, string, error) {
	meta := Metadata{
		scalars: map[string]string{},
		lists:   map[string][]string{},
	}

	if !strings.HasPrefix(document, "---\n") {
		return meta, document, nil
	}

	rest := document[len("---\n"):]
	endIdx := strings.Index(rest, "\n---\n")
	if endIdx < 0 {
		return meta, "", fmt.Errorf("unterminated front matter block")
	}
	block := rest[:endIdx]
	body := rest[endIdx+len("\n---\n"):]
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(block), &parsed); err != nil {
		return meta, "", fmt.Errorf("parse front matter yaml: %w", err)
	}
	for key, value := range parsed {
		switch typed := value.(type) {
		case string:
			meta.scalars[key] = typed
		case []any:
			for _, item := range typed {
				meta.lists[key] = append(meta.lists[key], fmt.Sprint(item))
			}
		case nil:
		default:
			meta.scalars[key] = fmt.Sprint(typed)
		}
	}

	return meta, body, nil
}
