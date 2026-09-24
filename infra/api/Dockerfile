# ---- runtime stage (scratch): zero dependências de imagem externa ----
# O binário é compilado no host (GOOS=linux CGO_ENABLED=0) e copiado para cá,
# evitando depender do pull de golang/alpine em redes instáveis.
# Build local:
#   $env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
#   go build -o bin/api-linux-amd64 ./cmd/api
FROM scratch
COPY bin/api-linux-amd64 /api

# O diretório de uploads precisa existir NA IMAGEM, já com o dono do processo.
# Ao montar um volume nomeado vazio sobre um caminho, o Docker inicializa o
# volume com o dono/permissões do diretório correspondente na imagem. Como a
# imagem `scratch` não tinha /uploads, o volume nascia root:root e a API (UID
# 65532) falhava ao gravar anexos e fotos de membro com "permission denied" —
# o erro só aparecia como "não foi possível salvar a imagem" na tela.
COPY --chown=65532:65532 infra/api/uploads/ /uploads/

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/api"]
