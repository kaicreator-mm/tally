-- +goose Up

-- 运维表:登录态(v3)。可随时清空,清了只是要重新登录。
-- 密码哈希放配置(x/crypto/bcrypt),不进数据库。
CREATE TABLE session (
  token      TEXT PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX session_expires_idx ON session (expires_at);

-- +goose Down
DROP TABLE IF EXISTS session;
