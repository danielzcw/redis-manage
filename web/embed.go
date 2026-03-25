package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var staticFiles embed.FS

func StaticFS() fs.FS {
	sub, _ := fs.Sub(staticFiles, "static")
	return sub
}
