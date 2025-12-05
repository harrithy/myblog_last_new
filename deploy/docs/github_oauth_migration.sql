-- GitHub OAuth 用户表字段迁移
-- 在 users 表中添加 GitHub 相关字段

-- 添加 github_id 字段（GitHub 用户唯一标识）
ALTER TABLE users ADD COLUMN github_id BIGINT UNIQUE;

-- 添加 avatar_url 字段（GitHub 头像 URL）
ALTER TABLE users ADD COLUMN avatar_url VARCHAR(500);

-- 添加 github_url 字段（GitHub 主页 URL）
ALTER TABLE users ADD COLUMN github_url VARCHAR(500);

-- 为 github_id 创建索引以加速查询
CREATE INDEX idx_users_github_id ON users(github_id);
