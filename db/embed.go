// Package db expõe as migrações SQL embutidas no binário via Go embed.
// Permite que a API aplique o schema versionado sem depender de arquivos externos.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS

// Setup embute o script de provisionamento dos papéis e privilégios padrão.
// Ele precisa rodar em CADA database criado, porque GRANT e
// ALTER DEFAULT PRIVILEGES têm escopo de database (não de cluster).
//
//go:embed init/setup.sql
var Setup string
