<#
.SYNOPSIS
    Inicializa o Chosen ERP (infra Docker + API Go + Webadmin Next.js).

.DESCRIPTION
    Arquivo ÚNICO de inicialização do projeto. Faz tudo, na ordem correta:
      1. Cria o .env a partir do .env.example (se não existir)
      2. Compila o binário Linux da API (necessário p/ imagem FROM scratch)
      3. Sobe os containers (postgres + api) e aguarda o healthcheck
      4. Instala dependências e inicia o Webadmin (se solicitado)

.PARAMETER Full
    Sobe também redis, rabbitmq, prometheus e grafana (docker compose --profile full).

.PARAMeter NoWeb
    Não inicia o Webadmin (sobe apenas a infraestrutura + API).

.PARAMETER Port
    Porta do Webadmin (padrão 33000).

.EXAMPLE
    .\start.ps1
    .\start.ps1 -Full
    .\start.ps1 -NoWeb
#>
param(
    [switch]$Full,
    [switch]$NoWeb,
    [int]$Port = 33000
)

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot

function Write-Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "    $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "    $msg" -ForegroundColor Yellow }

# ---------------------------------------------------------------- 1. ambiente
Write-Step "1/5 Preparando ambiente"
if (-not (Test-Path (Join-Path $root ".env"))) {
    Copy-Item (Join-Path $root ".env.example") (Join-Path $root ".env")
    Write-Ok ".env criado a partir de .env.example"
} else {
    Write-Ok ".env já existe"
}

# ------------------------------------------------------------ 2. binário Go
Write-Step "2/5 Compilando a API (binário Linux para a imagem)"
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
Write-Step "3/5 Subindo infraestrutura + API (Docker Compose)"
$compose = @("compose", "-f", (Join-Path $root "infra/docker-compose.yml"))
if ($Full) { $compose += @("--profile", "full") }
$compose += @("up", "-d", "--build")

Push-Location $root
try {
    & docker @compose
    if ($LASTEXITCODE -ne 0) { throw "Falha no docker compose up." }
} finally { Pop-Location }
Write-Ok "containers iniciados"

# --------------------------------------------------------- 4. health da API
Write-Step "4/5 Aguardando a API (http://localhost:38080/healthz)"
$ok = $false
for ($i = 0; $i -lt 40; $i++) {
    Start-Sleep -Seconds 2
    try {
        $r = Invoke-RestMethod -Uri "http://localhost:38080/healthz" -TimeoutSec 3
        if ($r.status -eq "ok") { $ok = $true; break }
    } catch { }
}
if (-not $ok) {
    Write-Warn "API ainda não respondeu. Verifique: docker logs chosen-api"
} else {
    Write-Ok "API saudável (as migrações já foram aplicadas no boot)"
}

# ------------------------------------------------------------- 5. webadmin
if (-not $NoWeb) {
    Write-Step "5/5 Iniciando o Webadmin (http://localhost:$Port)"
    $web = Join-Path $root "apps/webadmin"
    if (-not (Test-Path (Join-Path $web "node_modules"))) {
        Write-Host "    instalando dependências (npm install)..." -ForegroundColor Yellow
        Push-Location $web
        try { npm install } finally { Pop-Location }
    }
    Start-Process -FilePath "cmd.exe" -ArgumentList "/c", "cd /d $web && npm run dev -- -p $Port" -WindowStyle Minimized
    Write-Ok "Webadmin iniciado em segundo plano"
} else {
    Write-Host "`n==> 5/5 Webadmin ignorado (-NoWeb)" -ForegroundColor Cyan
}

# ------------------------------------------------------------------ resumo
Write-Host @"

============================================================
 Chosen ERP no ar
------------------------------------------------------------
 API (Go)        http://localhost:38080/healthz
 PostgreSQL      localhost:35432  (container interno 5432)
 Webadmin        http://localhost:$Port
$(if ($Full) { " Redis/RabbitMQ/Prometheus/Grafana (profile full) habilitados`n" })
 Login dev:      admin@demo.local / admin123          (Sede)
                 pastor.norte@demo.local / norte123   (Norte)
============================================================
"@ -ForegroundColor Green
