---
id: plataforma-backups
app: webadmin
title: Plataforma — backups e restauracao
category: Plataforma
summary: Gere, baixe, restaure e exclua pacotes completos de dados e arquivos por igreja.
routes: /dashboard/platform/backups
platform: true
tags: plataforma, backup, restore, restore, arquivos, retencao
order: 65
---

# Plataforma — backups e restauracao

O backup guarda os **dados completos** de uma igreja: tabelas, identidades, permissoes e arquivos de upload.

## Gerar

1. Abra **Plataforma > Backups** ou a aba **Backup** da igreja.
2. Clique em **Gerar backup**.
3. O pacote `.zip` contem `manifest.json`, `data.json` e `files/`.
4. Use **Baixar** para guardar uma copia externa.

## Restaurar

1. Selecione o pacote (por id ou upload `.zip`).
2. Confirme pelo slug da igreja.
3. O sistema gera um backup de seguranca (`pre_restore`) e substitui o conteudo daquela igreja.

As demais igrejas ficam intactas. A restauracao e destrutiva para o tenant alvo.

## Retencao

A configuracao global define `enabled`, `interval_hours` e `retention_days`. O worker gera backups das igrejas ativas e remove os antigos.

## Cuidados

- O pacote contem hashes de senha e segredos MFA: restrinja o acesso.
- Nao restaure em horario de uso intenso.
- Valide o backup antes de excluir o anterior.
