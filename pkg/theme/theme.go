package theme

import (
	"os"

	"blog-ssg/internal/domain/content"
	"gopkg.in/yaml.v3"
)

func Defaults() content.ThemeTokens {
	return content.ThemeTokens{
		Background:  "#ffffff",
		Text:        "#111111",
		Muted:       "#666666",
		Accent:      "#c75b3a",
		Surface:     "rgba(0, 0, 0, 0.05)",
		InverseText: "#f7f2ec",
		Overlay:     "#080808",
	}
}

func LoadConfig(path string) (content.TopicConfig, error) {
	cfg := content.TopicConfig{Theme: Defaults()}
	body, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	var raw struct {
		TopicTitle       string `yaml:"topic_title"`
		TopicHeader      string `yaml:"topic_header"`
		Background       string `yaml:"background"`
		Text             string `yaml:"text"`
		Muted            string `yaml:"muted"`
		Accent           string `yaml:"accent"`
		Surface          string `yaml:"surface"`
		InverseText      string `yaml:"inverse_text"`
		Overlay          string `yaml:"overlay"`
		LegacyHeading    string `yaml:"heading"`
		LegacyBorder     string `yaml:"border"`
		LegacyCodeBG     string `yaml:"code_bg"`
		LegacyCodeBorder string `yaml:"code_border"`
	}
	if err := yaml.Unmarshal(body, &raw); err != nil {
		return cfg, err
	}

	cfg.TopicTitle = raw.TopicTitle
	cfg.TopicHeader = raw.TopicHeader
	if raw.Background != "" {
		cfg.Theme.Background = raw.Background
	}
	if raw.Text != "" {
		cfg.Theme.Text = raw.Text
	}
	if raw.Muted != "" {
		cfg.Theme.Muted = raw.Muted
	}
	if raw.Accent != "" {
		cfg.Theme.Accent = raw.Accent
	}
	if raw.Surface != "" {
		cfg.Theme.Surface = raw.Surface
	}
	if raw.InverseText != "" {
		cfg.Theme.InverseText = raw.InverseText
	}
	if raw.Overlay != "" {
		cfg.Theme.Overlay = raw.Overlay
	}
	return cfg, nil
}
