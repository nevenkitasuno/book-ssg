# Repository Guidelines

## Project Structure & Module Organization

`cmd/ssg` contains the CLI entrypoint. Core orchestration lives in `internal/app/sitegen`. Domain types and invariants are in `internal/domain/*`. Filesystem loading and HTML rendering live in `internal/infra/contentloader` and `internal/infra/render`. Reusable helpers are in `pkg/*` (`frontmatter`, `preview`, `theme`, `assets`). Templates are embedded from `internal/infra/render/templates`. Example content fixtures live under `examples/content`. Design docs live under `docs/`.

## Build, Test, and Development Commands

- `go run ./cmd/ssg --content examples/content --output /private/tmp/blog-ssg-output` generates the sample site.
- `go test ./...` runs the full test suite.
- `go build ./cmd/ssg` builds the CLI binary.
- `gofmt -w <file>` formats changed Go files.

Use `examples/content` as the default local fixture unless a task explicitly needs a different content tree.

## Coding Style & Naming Conventions

Use standard Go formatting and keep files `gofmt`-clean. Prefer small packages with narrow responsibilities. Keep domain code free from CLI, filesystem, and HTML concerns. Use descriptive names: `Topic`, `Publication`, `PublicationPage`, `ThemeTokens`. Publication directories must follow `YYYY-MM-DD-slug`; page files must be `1.md`, `2.md`, etc.

For templates and CSS, preserve the current split: HTML in `.tmpl` files, shared styling in `base.css`, and theme tokens flowing through `Config.yaml`.

## Testing Guidelines

Tests use Go’s standard `testing` package. Keep tests next to the code they cover with `_test.go` suffix. Prefer table-driven tests for parsers and domain helpers. When changing generation behavior, run `go test ./...` and regenerate the sample site locally to inspect output paths and HTML structure.

## Commit & Pull Request Guidelines

Recent history uses short conventional-style subjects such as `feat: new header` and `feat: better hover shadow`. Follow that pattern: `feat: ...`, `fix: ...`, `docs: ...`, `refactor: ...`. Keep subjects imperative and concise.

PRs should describe the user-visible change, list validation commands run, and include screenshots or generated HTML snippets for template/CSS changes. If content structure or theme keys change, mention required updates to `examples/content/.../meta/Config.yaml`.
