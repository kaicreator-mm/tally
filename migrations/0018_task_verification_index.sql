-- +goose Up

-- P1-10:TaskContract / VerificationContract 的派生索引(PRD §33.3 Derived Index / INV-STATE-01)。
--
-- PRD §33.1:TaskContract / VerificationContract 的权威在 Git(版本化 desired 制品,
-- INV-ART-01;TaskContract 在 .tally/tasks/TASK-*.yaml);§33.2:PostgreSQL 只存
-- derived indexes。本迁移建 task_index / verification_index 两张索引面表,与 0017 的
-- project_index 同一性质:从 Git 制品快照派生、可整体 UPDATE/DELETE 重建,不是作者源。
--
-- 刻意不做的事:
--   - 无 append-only 触发器:派生索引必须可 UPDATE/DELETE 重建(INV-STATE-01),
--     与事实面(0001/0011)的只追加约束刻意相反;
--   - 不加任何 status 列:Task 的 lifecycle 状态是证据可追溯的派生事实(INV-STATE-02),
--     权威落点不在本索引;本表只回答「Git 里声明了哪个合同、哪个修订、在哪个路径、
--     当前投影是哪一行」(AC3:不引入权威 Task 状态列)。
--   - 不对 project_index 建外键:项目身份校验是 ingest 路径的职责
--     (§33.3 ghost-project guard,queries/project_index.sql 的 ProjectIndexExists),
--     与 0001 事实面的 TEXT project_id 惯例一致。
--
-- 行身份与列(AC1:project / id / revision / hash / path / current 投影):
--   project_id    项目身份,权威在 Git 注册表(project_index,0017)。
--   task_id       合同所属任务 id(如 TASK-023)。
--   revision/hash 修订标识,两个域的表示不同:
--     * task_index.revision:TaskContract 没有整数 revision;内容修订哈希
--       (internal/taskcontract 的 RevisionHash,sha256:<hex>)即其修订稳定标识
--       (hash.go:可直接被下游 execution/verification/trace 引用)。内容寻址:
--       内容不变则修订标识不变,内容任何实质变化都产生新行。
--     * verification_index:revision 是冻结契约声明的整数 revision
--       (verification.Contract.Revision,>= 1),hash 是冻结内容哈希
--       (verification.Frozen.Hash,sha256:<hex>)。二者共同标识一份冻结修订:
--       同号异容在 Git 历史上可能,不合并成一行。
--   path          Git 内制品路径(task:.tally/tasks/TASK-*.yaml),重建对账用。
--   is_current    当前投影标记:同一 (project_id, task_id) 至多一行 is_current
--                 (partial unique index 强制)。哪一版是当前投影由派生方按 Git
--                 快照声明,本表只存结论。
--   source_commit 派生依据的 Git 提交(Git 持有意图,重建按该快照对账)。
--   derived_at    本次派生时刻(重建/upsert 即刷新)。
--
-- 重建合同(INV-STATE-01,queries/task_index.sql / verification_index.sql):
-- 整体重建 = ClearTaskIndexCurrent/VerificationIndexCurrent(逐合同退位)→
-- Rebuild*(批量 upsert 快照)→ DeleteStale*(删除快照之外旧行),由派生方
-- 在一个事务内完成;重建后内容 = 快照,不残留手工/历史行。

CREATE TABLE task_index (
  project_id    TEXT NOT NULL,
  task_id       TEXT NOT NULL,
  revision      TEXT NOT NULL,                  -- 修订标识 = 内容修订哈希(sha256:<hex>,taskcontract.RevisionHash)
  path          TEXT NOT NULL,                  -- Git 内合同路径(.tally/tasks/TASK-*.yaml)
  is_current    BOOLEAN NOT NULL,               -- 当前投影:同一 (project_id, task_id) 至多一行
  source_commit TEXT NOT NULL,                  -- 派生自哪个 Git 提交
  derived_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, task_id, revision)
);

CREATE UNIQUE INDEX task_index_current_idx
  ON task_index (project_id, task_id) WHERE is_current;

CREATE TABLE verification_index (
  project_id    TEXT NOT NULL,
  task_id       TEXT NOT NULL,
  revision      INTEGER NOT NULL CHECK (revision >= 1), -- 冻结契约声明的 revision(verification.Contract.Revision)
  hash          TEXT NOT NULL,                  -- 冻结内容哈希(sha256:<hex>,verification.Frozen.Hash)
  path          TEXT NOT NULL,                  -- Git 内契约路径
  is_current    BOOLEAN NOT NULL,               -- 当前投影:同一 (project_id, task_id) 至多一行
  source_commit TEXT NOT NULL,                  -- 派生自哪个 Git 提交
  derived_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, task_id, revision, hash)
);

CREATE UNIQUE INDEX verification_index_current_idx
  ON verification_index (project_id, task_id) WHERE is_current;

-- +goose Down
DROP TABLE IF EXISTS verification_index;
DROP TABLE IF EXISTS task_index;
