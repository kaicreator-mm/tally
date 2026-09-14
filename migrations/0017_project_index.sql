-- +goose Up

-- P0-08:派生项目索引(PRD §33.3 Derived Index / §6.1 Git-first Artifact / INV-STATE-01)。
--
-- project_index 是索引面(派生数据):从 Git 项目注册表(.tally/project.yaml)派生,
-- 可整体删除重建,不是作者源。项目身份的权威在 Git;本表只是 DB 侧的已知项目清单,
-- 每行带 source_commit 记录派生自注册表的哪个提交,重算时按该快照对账。
--
-- 用途(ghost-project guard,§33.3):事实面五张表(0001)接受任意 project_id 文本,
-- 历史上留过「注册表里根本不存在的项目」的事实。此后所有 ingest 落库前先查本表
-- (queries/project_index.sql 的 ProjectIndexExists)验证 project identity,
-- 注册表之外的项目一律拒收。本表只提供判定,不回填、不改任何事实表。
--
-- 无 append-only 触发器:派生索引必须可 UPDATE/DELETE 重建(INV-STATE-01),
-- 与事实面的只追加约束(0001/0011)刻意相反;reindex 可随时清表重铺。

CREATE TABLE project_index (
  project_id    TEXT PRIMARY KEY,                   -- Git 注册表声明的项目身份
  source_commit TEXT NOT NULL,                      -- 派生自注册表的哪个 commit(Git 持有意图,重建依据)
  derived_at    TIMESTAMPTZ NOT NULL DEFAULT now()  -- 本次派生时刻(重建/upsert 即刷新)
);

-- +goose Down
DROP TABLE IF EXISTS project_index;
