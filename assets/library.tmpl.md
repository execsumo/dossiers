# Dossier Library

Harness: {{.Harness}}
Capabilities:
- MCP: {{if .Capabilities.MCP}}available{{else}}unavailable{{end}}
- Session-start hook: {{if .Capabilities.SessionStartHook}}available{{else}}unavailable{{end}}
- Session-end save hook: {{if .Capabilities.SessionEndHook}}available{{else}}unavailable{{end}}
- Pre-compaction save hook: {{if .Capabilities.PreCompactionHook}}available{{else}}unavailable{{end}}
- Transcript capture: {{if .Capabilities.TranscriptCapture}}available{{else}}unavailable{{end}}

{{if .Warnings}}
Warnings:
{{range .Warnings}}- {{.}}
{{end}}
{{end}}

Open Dossiers:
{{if .OpenDossiers}}
{{range .OpenDossiers}}- **{{.Name}}** (status: {{.Status}}, slug: {{.Slug}})
  Next Action: {{.NextAction}}
  Priority Score: {{.PriorityScore}}
{{end}}
{{else}}
(No open dossiers found)
{{end}}

Distillation Guide:
See: ~/.dossier/context/guide.md
