---
id: plataforma-igrejas
app: webadmin
title: Plataforma — igrejas
category: Plataforma
summary: Crie igrejas, edite dados/plano/limites/branding, gerencie usuarios de suporte e backups.
routes: /dashboard/platform/churches*
platform: true
tags: plataforma, igreja, tenant, plano, limites, branding, suporte, backup
order: 62
tour: administracao-plataforma
---

# Plataforma — igrejas

## Criar igreja

1. Abra **Plataforma > Igrejas**.
2. Clique em **Nova igreja**.
3. Informe nome, slug e plano.
4. O sistema cria a igreja, a Matriz, os papeis padrao e as permissoes.
5. Opcionalmente, crie o primeiro `super_admin`.

O subdominio `{slug}.erpchosen.com.br` passa a funcionar imediatamente.

## Gerenciar

Na ficha da igreja:

- **Dados**: nome, CNPJ, plano, limites e situacao.
- **Branding**: logo, cor, favicon e dominio proprio.
- **Usuarios**: suporte (listar/criar, definir perfis, ativar/desativar, redefinir senha).
- **Cobranca**: assinatura e faturas.
- **Backup**: gerar, baixar, restaurar e excluir pacotes.

## Suspender / reativar

Suspender define `is_active=false`. O login/refresh passa a responder `tenant_suspended`; a reativacao libera novamente.

## Boas praticas

- Valide o slug (minusculas, numeros e hifen).
- Use limites para evitar surpresa de quota.
- Antes de restaurar um backup, gere um backup de seguranca.
- Registre o motivo de suspensoes.
