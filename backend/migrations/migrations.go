// Package migrations embeds the SQL migration files shipped inside the binary.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
