-- 000078_plan_features.up.sql
-- ENTITLEMENTS POR PLANO: quais modulos cada igreja contratou.
--
-- `plans.features` (jsonb, ja existente) guarda o mapa por plano;
-- `tenants.features` (novo) guarda o OVERRIDE por igreja. A feature efetiva e o
-- plano sobreposto pelo override. Chave ausente = HABILITADA (retrocompativel:
-- igrejas existentes nao perdem modulo ate o admin configurar os planos).
--
-- Modulos core (people/admin/auth/app do membro) nao entram no catalogo e nunca
-- sao bloqueados. O enforcement e feito no backend (middleware), nao so no menu.

ALTER TABLE tenants ADD COLUMN features jsonb;

-- Matriz inicial dos planos. So preenche quem ainda esta vazio (nao sobrescreve
-- o que o operador ja ajustou no console).
UPDATE plans SET features = '{"finance":false,"reports":false,"governance":false,"ministries":true,"rosters":false,"events":true,"kids":false,"prayer":true,"whatsapp":false}'::jsonb
 WHERE key = 'starter' AND features = '{}'::jsonb;

UPDATE plans SET features = '{"finance":true,"reports":true,"governance":false,"ministries":true,"rosters":true,"events":true,"kids":true,"prayer":true,"whatsapp":true}'::jsonb
 WHERE key = 'pro' AND features = '{}'::jsonb;

UPDATE plans SET features = '{"finance":true,"reports":true,"governance":true,"ministries":true,"rosters":true,"events":true,"kids":true,"prayer":true,"whatsapp":true}'::jsonb
 WHERE key = 'enterprise' AND features = '{}'::jsonb;
