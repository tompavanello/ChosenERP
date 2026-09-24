-- 000023_member_marriage.up.sql
-- Data de casamento do membro, usada no relatório de aniversariantes
-- (aniversários de casamento) — requisito "Aniversariantes do mês (membros e
-- casamentos)" do módulo de relatórios.
--
-- Fica em `members` (como o endereço): é um dado da pessoa. Para casais, a
-- data tende a ser preenchida nos dois cônjuges; o relatório deduplica pelo
-- vínculo `spouse` para não listar o casal duas vezes.

ALTER TABLE members ADD COLUMN marriage_date date;
