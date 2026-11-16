//go:build desktop

package main

import (
	"embed"
	"io/fs"
)

//go:embed web
var bundledFrontend embed.FS

var frontendFiles, _ = fs.Sub(bundledFrontend, "web")
