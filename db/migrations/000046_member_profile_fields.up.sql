-- 000046_member_profile_fields.up.sql
-- Campos do cadastro de membro que existiam no legado (IPI) e nao tinham
-- destino no ChosenERP - sem eles a migracao perderia informacao:
--   * nationality -> ncad_naciona_2
--   * education   -> ncad_instruc_2 (escolaridade)
--   * notes       -> ncad_observa_2_o1..o20 (observacoes do cadastro)
ALTER TABLE members
    ADD COLUMN nationality text,
    ADD COLUMN education   text,
    ADD COLUMN notes       text;
