-- 000025_legacy_chart_of_accounts.down.sql
-- Remove o plano de contas legado e reativa as contas genéricas do seed.
UPDATE financial_categories SET is_active = true
WHERE code IN ('1.1','1.2','1.3','1.4','2.1','2.2','2.3','2.4','2.5');

UPDATE financial_classification_types SET is_active = true
WHERE code IN ('1.1','1.2','1.3','1.4','2.1','2.2','2.3','2.4','2.5');

DELETE FROM financial_classification_types
WHERE code IN ('101','102','103','104','105','106','107','108','109','110','111',
               '1','2','3','4','5','6','7','8','9','10','11','12','13','14','15','16',
               '17','18','19','20','21','22','23','24','25','26','27','28','29','30','31','32');

DELETE FROM financial_categories
WHERE code IN ('101','102','103','104','105','106','107','108','109','110','111',
               '1','2','3','4','5','6','7','8','9','10','11','12','13','14','15','16',
               '17','18','19','20','21','22','23','24','25','26','27','28','29','30','31','32');
