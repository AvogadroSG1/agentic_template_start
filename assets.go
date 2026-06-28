package mkproj

import (
	"embed"
	"io/fs"
)

var (
	//go:embed all:templates sources.yaml
	embeddedAssets embed.FS
)

func Assets() fs.FS {
	return embeddedAssets
}
