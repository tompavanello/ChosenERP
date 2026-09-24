-- 000023_member_marriage.up.sql
-- Data de casamento do membro, usada no relatorio de aniversariantes
-- (aniversarios de casamento) - requisito "Aniversariantes do mes (membros e
-- casamentos)" do modulo de relatorios.
--
-- Fica em `members` (como o endereco): e um dado da pessoa. Para casais, a
-- data tende a ser preenchida nos dois conjuges; o relatorio deduplica pelo
-- vinculo `spouse` para nao listar o casal duas vezes.

ALTER TABLE members ADD COLUMN marriage_date date;
