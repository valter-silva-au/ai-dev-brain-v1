package claude

import "embed"

// FS contains embedded template files
//
//go:embed *.md *.yaml
var FS embed.FS
