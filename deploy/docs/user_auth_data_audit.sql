-- User auth data audit
-- Purpose:
-- 1. Check whether historical users need cleanup for email + username + password login.
-- 2. Keep GitHub-only users visible in the audit without forcing unsafe changes.
-- 3. All active statements below are read-only SELECT queries.
--
-- Recommended before any manual cleanup:
-- CREATE TABLE users_backup_yyyymmdd AS SELECT * FROM users;

-- 1) High-level summary
SELECT
    COUNT(*) AS total_users,
    SUM(CASE WHEN TRIM(COALESCE(email, '')) = '' THEN 1 ELSE 0 END) AS missing_email,
    SUM(CASE WHEN TRIM(COALESCE(account, '')) = '' THEN 1 ELSE 0 END) AS missing_account,
    SUM(CASE WHEN TRIM(COALESCE(password, '')) = '' THEN 1 ELSE 0 END) AS missing_password,
    SUM(CASE WHEN TRIM(COALESCE(email, '')) <> '' AND LOWER(TRIM(email)) = LOWER(TRIM(COALESCE(account, ''))) THEN 1 ELSE 0 END) AS email_equals_account,
    SUM(CASE WHEN TRIM(COALESCE(account, '')) LIKE '%@%' THEN 1 ELSE 0 END) AS account_looks_like_email,
    SUM(CASE WHEN github_id IS NOT NULL AND github_id <> 0 THEN 1 ELSE 0 END) AS github_users
FROM users;

-- 2) Rows that are likely incomplete for the new login/register model
SELECT
    id,
    name,
    email,
    account,
    nickname,
    github_id,
    created_at
FROM users
WHERE TRIM(COALESCE(email, '')) = ''
   OR TRIM(COALESCE(account, '')) = ''
   OR TRIM(COALESCE(password, '')) = ''
ORDER BY id ASC;

-- 3) Duplicate emails after trimming/lowercasing
SELECT
    LOWER(TRIM(email)) AS normalized_email,
    COUNT(*) AS duplicate_count,
    GROUP_CONCAT(id ORDER BY id SEPARATOR ',') AS user_ids
FROM users
WHERE TRIM(COALESCE(email, '')) <> ''
GROUP BY LOWER(TRIM(email))
HAVING COUNT(*) > 1
ORDER BY duplicate_count DESC, normalized_email ASC;

-- 4) Duplicate usernames after trimming/lowercasing
SELECT
    LOWER(TRIM(account)) AS normalized_account,
    COUNT(*) AS duplicate_count,
    GROUP_CONCAT(id ORDER BY id SEPARATOR ',') AS user_ids
FROM users
WHERE TRIM(COALESCE(account, '')) <> ''
GROUP BY LOWER(TRIM(account))
HAVING COUNT(*) > 1
ORDER BY duplicate_count DESC, normalized_account ASC;

-- 5) Users where email and username are still the same value
SELECT
    id,
    name,
    email,
    account,
    nickname,
    github_id,
    created_at
FROM users
WHERE TRIM(COALESCE(email, '')) <> ''
  AND LOWER(TRIM(email)) = LOWER(TRIM(COALESCE(account, '')))
ORDER BY id ASC;

-- 6) Users whose username still looks like an email address
SELECT
    id,
    name,
    email,
    account,
    nickname,
    github_id,
    created_at
FROM users
WHERE TRIM(COALESCE(account, '')) LIKE '%@%'
ORDER BY id ASC;

-- 7) GitHub users without a local password
-- These are usually fine if you want them to remain GitHub-login-only.
SELECT
    id,
    name,
    email,
    account,
    nickname,
    github_id,
    created_at
FROM users
WHERE github_id IS NOT NULL
  AND github_id <> 0
  AND TRIM(COALESCE(password, '')) = ''
ORDER BY id ASC;

-- 8) Local users without a password
-- These are the highest-risk records if you expect password login for every non-GitHub account.
SELECT
    id,
    name,
    email,
    account,
    nickname,
    github_id,
    created_at
FROM users
WHERE (github_id IS NULL OR github_id = 0)
  AND TRIM(COALESCE(password, '')) = ''
ORDER BY id ASC;

-- Optional remediation examples (commented out on purpose)
-- Review carefully before using any UPDATE statements.

-- Example A: assign a guaranteed-unique fallback username to rows with a blank account.
-- UPDATE users
-- SET account = CONCAT('user_', id)
-- WHERE TRIM(COALESCE(account, '')) = '';

-- Example B: assign a guaranteed-unique fallback username to rows where account still equals email.
-- UPDATE users
-- SET account = CONCAT('user_', id)
-- WHERE TRIM(COALESCE(email, '')) <> ''
--   AND LOWER(TRIM(email)) = LOWER(TRIM(COALESCE(account, '')));

-- Example C: do NOT auto-fill missing passwords here.
-- For local users with empty passwords, prefer a password reset flow or manual reset.
