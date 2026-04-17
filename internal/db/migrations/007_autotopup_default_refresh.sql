ALTER TABLE users ALTER COLUMN autotopup_threshold_cents SET DEFAULT 1000000;
ALTER TABLE users ALTER COLUMN autotopup_amount_cents SET DEFAULT 2000000;

UPDATE users
SET
    autotopup_threshold_cents = 1000000,
    autotopup_amount_cents = 2000000,
    updated_at = now()
WHERE
    autotopup_enabled = FALSE
    AND autotopup_threshold_cents = 50000
    AND autotopup_amount_cents = 100000;
