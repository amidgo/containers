package testmigrations

import (
	"embed"
)

//go:embed *.sql
var migrations embed.FS

func Embed() embed.FS {
	return migrations
}
