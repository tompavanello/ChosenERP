// Package db expoe as migracoes SQL embutidas no binario via Go embed.
// Permite que a API aplique o schema versionado sem depender de arquivos externos.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS

// Setup embute o script de provisionamento dos papeis e privilegios padrao.
// Ele precisa rodar em CADA database criado, porque GRANT e
// ALTER DEFAULT PRIVILEGES tem escopo de database (nao de cluster).
//
//go:embed init/setup.sql
var Setup string
