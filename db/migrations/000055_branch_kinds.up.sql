-- 000055_branch_kinds.up.sql
-- Reclassifica o tipo de filial para a estrutura de governo da igreja:
--   * matriz — Sede Administrativa (templo principal; controla as demais).
--   * filial — Igreja Local / Congregação (estabelecida, caixa próprio).
--   * pae    — Ponto de Atendimento de Evangelização (núcleo inicial, sem
--              autonomia; OBRIGATORIAMENTE vinculado a uma Matriz ou Filial).
--
-- Mapeamento dos valores antigos:
--   branch + raiz (parent NULL) -> matriz
--   branch + com pai            -> filial
--   congregation                -> filial
--   sub_congregation + com pai  -> pae
--   sub_congregation + sem pai  -> filial (não pode ser PAE sem supervisor)

-- 1) Normalização dos dados existentes (antes de fechar o domínio).
UPDATE branches SET kind = CASE
    WHEN kind = 'matriz' THEN 'matriz'
    WHEN kind = 'filial' THEN 'filial'
    WHEN kind = 'pae'    THEN 'pae'
    WHEN kind = 'congregation' THEN 'filial'
    WHEN kind = 'sub_congregation' THEN
        CASE WHEN parent_id IS NULL THEN 'filial' ELSE 'pae' END
    WHEN kind = 'branch' THEN
        CASE WHEN parent_id IS NULL THEN 'matriz' ELSE 'filial' END
    ELSE CASE WHEN parent_id IS NULL THEN 'matriz' ELSE 'filial' END
END;

-- 2) O tipo padrão de uma nova unidade passa a ser Filial.
ALTER TABLE branches ALTER COLUMN kind SET DEFAULT 'filial';

-- 3) Domínio fechado + regras estruturais declarativas.
ALTER TABLE branches DROP CONSTRAINT IF EXISTS branches_kind_check;
ALTER TABLE branches ADD CONSTRAINT branches_kind_check
    CHECK (kind IN ('matriz', 'filial', 'pae'));

ALTER TABLE branches DROP CONSTRAINT IF EXISTS branches_matriz_root_check;
ALTER TABLE branches ADD CONSTRAINT branches_matriz_root_check
    CHECK (kind <> 'matriz' OR parent_id IS NULL);

ALTER TABLE branches DROP CONSTRAINT IF EXISTS branches_pae_parent_check;
ALTER TABLE branches ADD CONSTRAINT branches_pae_parent_check
    CHECK (kind <> 'pae' OR parent_id IS NOT NULL);

-- 4) PAE só pode ser supervisionado por Matriz ou Filial (regra entre linhas).
CREATE OR REPLACE FUNCTION branches_validate_hierarchy() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    parent_kind text;
BEGIN
    IF NEW.kind = 'matriz' AND NEW.parent_id IS NOT NULL THEN
        RAISE EXCEPTION 'Matriz não pode ter filial superior' USING ERRCODE = '23514';
    END IF;

    IF NEW.kind = 'pae' THEN
        IF NEW.parent_id IS NULL THEN
            RAISE EXCEPTION 'PAE precisa estar vinculado a uma Matriz ou Filial' USING ERRCODE = '23514';
        END IF;
        SELECT kind INTO parent_kind FROM branches WHERE id = NEW.parent_id;
        IF parent_kind IS NULL THEN
            RAISE EXCEPTION 'Superior do PAE inexistente' USING ERRCODE = '23514';
        END IF;
        IF parent_kind NOT IN ('matriz', 'filial') THEN
            RAISE EXCEPTION 'PAE só pode ser supervisionado por Matriz ou Filial' USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS branches_validate_hierarchy ON branches;
CREATE TRIGGER branches_validate_hierarchy
    BEFORE INSERT OR UPDATE ON branches
    FOR EACH ROW EXECUTE FUNCTION branches_validate_hierarchy();
