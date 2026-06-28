package assets

import "embed"

// FS holds the embedded assets for Dossier (Distillation Guide and context templates).
//
//go:embed guide.md library.tmpl.md instructions.md
var FS embed.FS
