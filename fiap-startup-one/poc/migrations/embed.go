// Package migrations embute as migrações SQL no binário.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
