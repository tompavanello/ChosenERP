-- 000070_programacao_title.up.sql
-- Titulo amigavel para a programacao de tipo "outro" (ex.: "Ensaio de teatro",
-- "Vigilia de Avivamento"). O titulo e propagado para a ocorrencia na agenda
-- (`church_events.title`), que passa a exibi-lo no lugar do rotulo generico
-- "Outro". Quando o tipo nao e "outro", o titulo fica vazio e vale o nome do
-- tipo no catalogo selado.

ALTER TABLE programacoes ADD COLUMN title text;
ALTER TABLE church_events ADD COLUMN title text;
