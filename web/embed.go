// Package web embeds the HTML templates and static assets into the binary
// so the compiled server carries them with no on-disk dependency (important
// for the Docker image, which copies only the binary).
package web

import "embed"

//go:embed templates/*.html
var TemplatesFS embed.FS

//go:embed static
var StaticFS embed.FS
