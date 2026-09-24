-- 000049_reciprocal_relationships.up.sql
-- Backfill dos vinculos de membro para o formato reciproco: cada linha
-- member_id -> related_id ganha a linha inversa related_id -> member_id, com o
-- kind invertido (pai<->filho, discipulador<->discipulo; conjuge/parente/
-- dependente sao simetricos). Sem isso, a referencia ficaria correta so na ficha
-- de um lado. Idempotente (ON CONFLICT) e nao toca auto-vinculos.
INSERT INTO member_relationships (member_id, related_id, kind)
SELECT r.related_id, r.member_id,
       CASE r.kind
           WHEN 'parent' THEN 'child'
           WHEN 'child' THEN 'parent'
           WHEN 'discipler' THEN 'disciple'
           WHEN 'disciple' THEN 'discipler'
           ELSE r.kind
       END
FROM member_relationships r
WHERE r.member_id <> r.related_id
ON CONFLICT (member_id, related_id, kind) DO NOTHING;
