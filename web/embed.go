package web

import (
	"embed"
	"io/fs"
)

// TemplatesFS embeds all HTML templates for the UI.
//go:embed templates/*.html
var templatesFS embed.FS

// StaticFS embeds static assets such as JS and CSS.
//go:embed static/*
var staticFS embed.FS

// Templates returns the filesystem for HTML templates.
func Templates() fs.FS {
	return templatesFS
}

// Static returns the filesystem for static assets.
// It exposes the embedded "static" directory at the FS root so that
// requests to "/static/..." map to files like "htmx.min.js".
func Static() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return staticFS
	}
	return sub
}

