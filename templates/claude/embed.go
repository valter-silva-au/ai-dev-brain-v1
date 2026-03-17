package claude

import "embed"

// FS contains embedded template files
//
//go:embed *.md *.yaml *.sh
var FS embed.FS
