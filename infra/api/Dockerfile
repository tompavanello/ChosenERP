# ---- runtime stage (scratch): zero dependências de imagem externa ----
# O binário é compilado no host (GOOS=linux CGO_ENABLED=0) e copiado para cá,
# evitando depender do pull de golang/alpine em redes instáveis.
# Build local:
#   $env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
#   go build -o bin/api-linux-amd64 ./cmd/api
FROM scratch
COPY bin/api-linux-amd64 /api
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/api"]
