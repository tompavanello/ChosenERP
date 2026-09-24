<#
.SYNOPSIS
    Inicializa o Chosen ERP (infra Docker + API Go + Webadmin Next.js).

.DESCRIPTION
    Arquivo UNICO de inicializacao do projeto. Faz tudo, na ordem correta:
       1. Cria o .env a partir do .env.example (se nao existir)
       2. Compila o binario Linux da API (necessario p/ imagem FROM scratch)
       3. Sobe os containers (postgres + api + webadmin + nginx) e aguarda o healthcheck
       4. Verifica a saude da API e do Nginx

.PARAMETER Full
    Sobe tambem redis, rabbitmq, prometheus e grafana (docker compose --profile full).

.PARAMETER Port
    Porta do Webadmin exibida no resumo (padrao 33000; o Compose usa WEBADMIN_PORT).

.EXAMPLE
    .\start.ps1
    .\start.ps1 -Full
#>
param(
    [switch]$Full,
    [int]$Port = 33000
)

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot

function Write-Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "    $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "    $msg" -ForegroundColor Yellow }

# ---------------------------------------------------------------- 1. ambiente
Write-Step "1/4 Preparando ambiente"
if (-not (Test-Path (Join-Path $root ".env"))) {
    Copy-Item (Join-Path $root ".env.example") (Join-Path $root ".env")
    Write-Ok ".env criado a partir de .env.example"
} else {
    Write-Ok ".env ja existe"
}

# ------------------------------------------------------------ 2. binario Go
Write-Step "2/4 Compilando a API (binario Linux para a imagem)"
$env:GOOS = "linux"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
Push-Location $root
try {
    go build -o (Join-Path $root "bin/api-linux-amd64") ./cmd/api
    if ($LASTEXITCODE -ne 0) { throw "Falha ao compilar a API (go build)." }
    Write-Ok "bin/api-linux-amd64 gerado"
} finally {
    Remove-Item Env:\GOOS, Env:\GOARCH, Env:\CGO_ENABLED -ErrorAction SilentlyContinue
    Pop-Location
}

# ------------------------------------------------------------- 3. containers
Write-Step "3/4 Subindo infraestrutura + API + Nginx (Docker Compose)"
# O --env-file e obrigatorio: com `-f infra/docker-compose.yml` o project
# directory passa a ser `infra/`, e o Compose procura o .env LA - nunca na
# raiz, onde este script o cria. Sem a flag, todo valor do .env e ignorado em
# silencio e o Compose cai nos defaults do proprio arquivo. Como os defaults
# do docker-compose.yml coincidem com o .env.example, isso passou despercebido
# ate alguem precisar de um valor diferente (o JWT_SECRET).
$compose = @("compose", "--env-file", (Join-Path $root ".env"), "-f", (Join-Path $root "infra/docker-compose.yml"))
if ($Full) { $compose += @("--profile", "full") }
$compose += @("up", "-d", "--build")

Push-Location $root
try {
    & docker @compose
    if ($LASTEXITCODE -ne 0) { throw "Falha no docker compose up." }
} finally { Pop-Location }
Write-Ok "containers iniciados"

# --------------------------------------------------------- 4. health
Write-Step "4/4 Aguardando API e Nginx (http://localhost/healthz)"
$ok = $false
for ($i = 0; $i -lt 40; $i++) {
    Start-Sleep -Seconds 2
    try {
        $r = Invoke-RestMethod -Uri "http://localhost:38080/healthz" -TimeoutSec 3
        if ($r.status -eq "ok") { $ok = $true; break }
    } catch { }
}
if (-not $ok) {
    Write-Warn "API ainda nao respondeu. Verifique: docker logs chosen-api"
} else {
    Write-Ok "API saudavel (as migracoes ja foram aplicadas no boot)"
}

# Verificar nginx
$nginxOk = $false
for ($i = 0; $i -lt 20; $i++) {
    Start-Sleep -Seconds 1
    try {
        $r = Invoke-RestMethod -Uri "http://localhost/healthz" -TimeoutSec 3
        if ($r.status -eq "ok") { $nginxOk = $true; break }
    } catch { }
}
if (-not $nginxOk) {
    Write-Warn "Nginx nao respondeu. Verifique: docker logs chosen-nginx"
} else {
    Write-Ok "Nginx funcionando (proxy para a API)"
}

# ------------------------------------------------------------- 5. webadmin
# O webadmin agora sobe DENTRO do Docker Compose (servico `webadmin`, imagem
# standalone, porta ${WEBADMIN_PORT:-33000}), com `restart: unless-stopped`.
# Nada e iniciado no host: ao ligar o Docker, api + webadmin + nginx voltam.
Write-Step "5/5 Webadmin no Docker (http://localhost:$Port)"
Write-Ok "servico 'webadmin' publicado em http://localhost:$Port"

# ------------------------------------------------------------------ resumo
Write-Host @"

 ============================================================
  Chosen ERP no ar
 ------------------------------------------------------------
  Webadmin       http://localhost:$Port
  Nginx (proxy)  http://localhost/
  API (Go)       http://localhost:38080/healthz
  PostgreSQL     localhost:35432  (container interno 5432)
 $(if ($Full) { " Redis/RabbitMQ/Prometheus/Grafana (profile full) habilitados`n" })
  Login dev:      admin@demo.local          (Sede)
                  pastor.norte@demo.local   (Norte)
                  senhas em .env -> DEMO_ADMIN_PASSWORD / DEMO_NORTE_PASSWORD
 ============================================================
"@ -ForegroundColor Green
