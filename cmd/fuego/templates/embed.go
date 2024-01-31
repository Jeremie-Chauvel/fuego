package templates

import (
	"embed"
)

//go:embed *.template.go
var FS embed.FS
