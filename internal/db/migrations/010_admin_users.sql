ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users
SET is_admin = TRUE,
    updated_at = now()
WHERE lower(email) = 'leepenkman@gmail.com';
