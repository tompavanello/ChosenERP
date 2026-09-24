# Cloudflare Tunnel do `erpchosen.com.br`

Tunnel **exclusivo** do white-label (apex, `www` e `*.erpchosen.com.br`),
separado do tunnel que serve `mgmconsultoria.com`. Ele roda como o servico
`cloudflared-erpchosen` do Compose (profile `erpchosen`) e entrega tudo para o
nginx (`http://nginx:80`), que faz o split `/api` -> API e o resto -> webadmin,
repassando `X-Tenant-Slug` conforme o subdominio.

Arquivos:
- `config-erpchosen.yml` - ingress do tunnel (commitado; so tem o id do tunnel).
- `erpchosen-creds.json` - credenciais do tunnel (**segredo**, ignorado no git).

## Como foi criado (referencia)

```
# login na conta dona do erpchosen.com.br (abre o navegador)
cloudflared tunnel login
cloudflared tunnel create erpchosen
cloudflared tunnel route dns <tunnel-id> erpchosen.com.br
cloudflared tunnel route dns <tunnel-id> www.erpchosen.com.br
cloudflared tunnel route dns <tunnel-id> "*.erpchosen.com.br"
```

Depois: copiar o JSON de credenciais para `infra/cloudflare/erpchosen-creds.json`
e ajustar o `config-erpchosen.yml` com o tunnel id.

## Subir / atualizar

```
docker compose --env-file .env -f infra/docker-compose.yml \
  --profile erpchosen up -d cloudflared-erpchosen
```

O container tem `restart: unless-stopped`, entao volta sozinho no boot (desde que
nao seja removido).

## Subdominio por igreja

O subdominio e `{tenant.slug}.erpchosen.com.br` (ex.: `demo.erpchosen.com.br`).
O wildcard cobre qualquer slug automaticamente; a API valida que a identidade tem
vinculo ativo com a igreja do slug no login.
