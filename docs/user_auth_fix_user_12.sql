-- Targeted remediation for historical user id=12
-- Before running in any shared environment, back up the row:
-- SELECT * FROM users WHERE id = 12;

START TRANSACTION;

UPDATE users
SET
    account = 'testuser',
    password = '$2a$10$LHq4WxekFmgQLig7vyJJte1GohHl2AVVZvvCl3mGlA2IMsp2uGbFC'
WHERE id = 12
  AND LOWER(TRIM(COALESCE(email, ''))) = LOWER('test@example.com');

SELECT id, name, email, account, nickname, github_id, created_at
FROM users
WHERE id = 12;

COMMIT;

-- Temporary password for the hash above:
-- TempPass#20260324!
--
-- Recommended follow-up:
-- 1. Log in once to confirm the account works.
-- 2. Change the password through a reset/update flow when available.
