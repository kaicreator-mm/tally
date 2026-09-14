-- +goose Up

-- 复现上下文的权威来源(T11b / D3 正式答案):诊断要能真正复现,得知道
-- 在哪个目录、用什么命令重跑那次构建。这些是失败自带的属性,录入时不记、
-- 事后无处还原,所以随失败一起持久化进事实面(只加列,只随 INSERT 落库,INV-5)。
--
-- 三列 NOT NULL DEFAULT:存量行自动补默认值(空),旧失败仍走
-- 「Handle 缺 Repro 跳过 G1/G2」路径,行为不回归(D3 语义保留)。
ALTER TABLE failure_event ADD COLUMN workdir TEXT NOT NULL DEFAULT '';
ALTER TABLE failure_event ADD COLUMN repro TEXT NOT NULL DEFAULT '';
ALTER TABLE failure_event ADD COLUMN other_targets JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE failure_event DROP COLUMN other_targets;
ALTER TABLE failure_event DROP COLUMN repro;
ALTER TABLE failure_event DROP COLUMN workdir;
