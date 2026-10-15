# ---- stage de certificados: bundle CA para chamadas HTTPS ----
# A imagem final e `scratch` (sem sistema), entao o Go nao teria raizes de CA e
# qualquer HTTPS falharia ("x509: certificate signed by unknown authority").
# Isso quebrava api.bible, Asaas e WhatsApp Meta. O alpine:3.20 (ja usado no
# stack) traz o bundle pronto; copiamos para o caminho que o Go procura.
FROM alpine:3.20 AS certs

# ---- runtime stage (scratch): zero dependencias de imagem externa ----
# O binario e compilado no host (GOOS=linux CGO_ENABLED=0) e copiado para ca,
# evitando depender do pull de golang/alpine em redes instaveis.
# Build local:
#   $env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
#   go build -o bin/api-linux-amd64 ./cmd/api
FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY bin/api-linux-amd64 /api

# O diretorio de uploads precisa existir NA IMAGEM, ja com o dono do processo.
# Ao montar um volume nomeado vazio sobre um caminho, o Docker inicializa o
# volume com o dono/permissões do diretório correspondente na imagem. Como a
# imagem `scratch` não tinha /uploads, o volume nascia root:root e a API (UID
# 65532) falhava ao gravar anexos e fotos de membro com "permission denied" —
# o erro só aparecia como "não foi possível salvar a imagem" na tela.
COPY --chown=65532:65532 infra/api/uploads/ /uploads/

# Mesmo problema para o volume de backups: o diretorio precisa existir na
# imagem com o dono do processo para o volume nomeado `backups` nascer gravavel.
COPY --chown=65532:65532 infra/api/backups/ /backups/

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/api"]
