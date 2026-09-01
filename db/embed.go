// Package db expõe as migrações SQL embutidas no binário via Go embed.
// Permite que a API aplique o schema versionado sem depender de arquivos externos.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
