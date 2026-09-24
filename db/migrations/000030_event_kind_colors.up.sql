-- 000030_event_kind_colors.up.sql
-- Cor por tipo de evento (exibida no calendário).

ALTER TABLE event_kinds ADD COLUMN color text;

UPDATE event_kinds SET color = CASE slug
    WHEN 'escola_biblica_dominical' THEN '#10b981'
    WHEN 'culto'                    THEN '#0ea5e9'
    WHEN 'culto_especial'           THEN '#6366f1'
    WHEN 'reuniao'                  THEN '#f59e0b'
    WHEN 'pequeno_grupo'            THEN '#8b5cf6'
    WHEN 'evento_jovens'            THEN '#ec4899'
    WHEN 'escola_biblica'           THEN '#14b8a6'
    WHEN 'santa_ceia'               THEN '#ef4444'
    WHEN 'vigilia'                  THEN '#64748b'
    ELSE '#94a3b8'
END
WHERE color IS NULL;
