-- +goose Up

-- P1-12:VerificationRun / Evidence / Exposure 持久化合同。
-- PRD §6.4.3 / §8.13:Contract、Run、Evidence 三者分离 —— VerificationContract
-- 定义「应该验证什么」(desired,Git 制品),VerificationRun 是某 contract revision
-- 对某 immutable ChangeSet 的一次实际执行,VerificationEvidence 是该次执行产生的
-- observed 事实。Exposure 是 Fact,不是 Contract mutation(PRD §8.6.1):held-out
-- 内容/verdict 进入实现侧可见通道时只追加暴露事实,绝不回写 VerificationContract。
--
-- INV-FACT-01:四张表全部只追加 —— 复用 0001 的 tally_prevent_modify,并按 0011
-- 的结论直接建成语句级触发器(行级触发器在 0 行匹配的 UPDATE/DELETE 上不触发,
-- 堵不住「没碰巧改到东西」的篡改语句)。历史 Run 全量保留(§8.13),同一
-- (contract revision, ChangeSet) 的多次执行各自成行,禁止 cherry-pick 靠的正是
-- 全量留痕(§8.7.2)。
--
-- 有效性(validity)与适用性(applicability)是两条独立的轴(PRD §8.8.1/8.8.2):
--   - evidence 行诞生即 valid;失效只能向 verification_evidence_ruling 追加带
--     provenance 的裁定,本表任何行都不改写(合法 FAIL 不被后来的 PASS 抹掉);
--   - applicability 由行上的精确 scope 引用(project/task/acceptance/
--     contract revision/ChangeSet)相对当前 desired 上下文判定。
--
-- 跨任务边界:P1-10 拥有契约派生索引(0018)、P1-11 拥有 CodegenRun/ChangeSet
-- 快照表(0019)。身份是内容派生的不透明串(verification.Frozen.Hash、
-- changeset.ChangeSetID),因此本文件按值引用它们,不建跨任务外键:各分支
-- 独立可迁移,合并不互相阻塞。evidence→run 的外键指向本文件内的表,不受此限。

-- verification_run:一次针对 immutable ChangeSet 的验证执行(PRD §8.7.1:verdict
-- 绑定 immutable candidate + contract revision + frozen protocol + 环境身份)。
CREATE TABLE verification_run (
  id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id          TEXT NOT NULL,
  task_id             TEXT NOT NULL,
  contract_revision   INTEGER NOT NULL CHECK (contract_revision >= 1),
  contract_hash       TEXT NOT NULL,                -- 冻结 VerificationContract 内容哈希(sha256:<hex>)
  changeset_id        TEXT NOT NULL,                -- immutable ChangeSet 身份(内容派生,值引用 P1-11 快照面)
  env_name            TEXT NOT NULL,                -- 验证环境身份 = 名称 + 定义内容哈希(挂内容哈希,不挂逻辑号)
  env_definition_hash TEXT NOT NULL,
  version_combo       JSONB NOT NULL,               -- 当时环境版本组合,冗余自包含(同 0001 约定)
  runtime             TEXT NOT NULL,                -- 执行后端身份(FakeRuntime/…)
  started_at          TIMESTAMPTZ NOT NULL,
  finished_at         TIMESTAMPTZ,
  status              TEXT NOT NULL CHECK (status IN ('pass','fail','error'))
    -- 落行时一次写入的观测结果;生命周期 VERIFIED 等状态另行派生(INV-STATE-01),
    -- 本列不是可变任务状态,表本身只追加。
);
CREATE TRIGGER verification_run_append_only
  BEFORE UPDATE OR DELETE ON verification_run
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

-- verification_evidence:一次 Run 产生的 observed evidence 事实(PRD §6.4.3、§14.3:
-- 结构化 command、exit code、stdout/stderr digest、runtime、validator version、
-- 环境身份等才是 Evidence;Agent 的「我认为通过」只是 Claim,进不了本表 ——
-- recorder_kind 枚举里没有 implementation_agent,INV-AUTH-01/02)。
CREATE TABLE verification_evidence (
  id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  fact_id           TEXT NOT NULL UNIQUE,           -- 证据事实身份,唯一且永不复用;重复记录=错误,不是更新
  run_id            BIGINT NOT NULL REFERENCES verification_run(id),
  project_id        TEXT NOT NULL,
  task_id           TEXT NOT NULL,
  acceptance_id     TEXT NOT NULL,                  -- applicability 引用:验收条款
  binding_id        TEXT NOT NULL,                  -- applicability 引用:契约内 evidence binding
  visibility_policy TEXT NOT NULL CHECK (visibility_policy IN ('visible','held_out')),
    -- 冻结契约策略在执行时的快照(PRD §8.4.1:policy 冻结后不因运行事实改写)
  verdict           TEXT NOT NULL CHECK (verdict IN ('pass','fail','error')),
  recorder_kind     TEXT NOT NULL CHECK (recorder_kind IN ('validator','verification_run','human','external_system')),
    -- 刻意排除 implementation_agent:实现侧通道不能确认机器事实(INV-AUTH-01/02)
  recorder_actor    TEXT NOT NULL,                  -- validator 名/run 引用/人工身份/外部系统引用
  validator_version TEXT,
  command           JSONB CHECK (command IS NULL OR jsonb_typeof(command) = 'object'),
    -- 结构化 CommandSpec 规范 JSON(0016 合同:argv/cwd/env/timeout/shell),拒绝 shell 字符串
  exit_code         INTEGER,
  stdout_digest     TEXT,
  stderr_digest     TEXT,
  raw_output_ref    TEXT,                           -- 完整输出在 Tally 控制存储中的引用;held_out 的原始失败输出不得进实现侧可见通道(INV-VER-05、§14.3)
  recorded_at       TIMESTAMPTZ NOT NULL
);
CREATE TRIGGER verification_evidence_append_only
  BEFORE UPDATE OR DELETE ON verification_evidence
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

-- verification_evidence_ruling:对 evidence 的失效裁定,只追加(PRD §8.8.1:
-- failed evidence 只有经 confirmed holdout_defect / SpecDefect / invalid-evidence
-- 等带 provenance 的正式裁定后才失去 validity)。kind 只收三类失效裁定;
-- 裁定通道本身必须 fact-capable(与 evidence 同一约束)。
CREATE TABLE verification_evidence_ruling (
  id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  evidence_id    BIGINT NOT NULL REFERENCES verification_evidence(id),
  project_id     TEXT NOT NULL,
  kind           TEXT NOT NULL CHECK (kind IN ('holdout_defect','spec_defect','invalid_evidence')),
  reason         TEXT NOT NULL,
  recorder_kind  TEXT NOT NULL CHECK (recorder_kind IN ('validator','verification_run','human','external_system')),
  recorder_actor TEXT NOT NULL,
  ruled_at       TIMESTAMPTZ NOT NULL
);
CREATE TRIGGER verification_evidence_ruling_append_only
  BEFORE UPDATE OR DELETE ON verification_evidence_ruling
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

-- evidence_exposure:held-out 内容或诊断进入 Implementation Authority 可见通道的
-- append-only Exposure Fact(PRD §8.6)。channel 是开放集合,§8.6.1 只给典型值
-- (debug_disclosure/session_leak/validation_log/knowledge_index/owner_disclosure/
-- manual),故不做枚举 CHECK。引用三件套:
--   - binding 引用:contract_revision + contract_hash + binding_id(可加 acceptance_id);
--   - lineage 引用:changeset_id 可空 —— 内容暴露可能先于任何具体 candidate 被观测;
--   - channel:进入的可见通道。
-- 本表只记暴露事实与引用:detail 不得携带 held-out 内容或可识别 verdict(INV-VER-05)。
CREATE TABLE evidence_exposure (
  id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id        TEXT NOT NULL,
  task_id           TEXT NOT NULL,
  contract_revision INTEGER NOT NULL CHECK (contract_revision >= 1),
  contract_hash     TEXT NOT NULL,
  binding_id        TEXT NOT NULL,
  acceptance_id     TEXT,
  changeset_id      TEXT,
  channel           TEXT NOT NULL,
  detail            JSONB CHECK (detail IS NULL OR jsonb_typeof(detail) = 'object'),
  exposed_at        TIMESTAMPTZ NOT NULL
);
CREATE TRIGGER evidence_exposure_append_only
  BEFORE UPDATE OR DELETE ON evidence_exposure
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

-- 热路径索引(同 0001:只建现在就要的)。
CREATE INDEX verification_run_changeset_idx ON verification_run (project_id, changeset_id, started_at DESC);
CREATE INDEX verification_run_task_idx ON verification_run (project_id, task_id, started_at DESC);
CREATE INDEX verification_evidence_run_idx ON verification_evidence (run_id);
CREATE INDEX verification_evidence_scope_idx ON verification_evidence (project_id, task_id, acceptance_id);
CREATE INDEX verification_evidence_ruling_evidence_idx ON verification_evidence_ruling (evidence_id);
CREATE INDEX evidence_exposure_binding_idx ON evidence_exposure (project_id, contract_hash, binding_id);
CREATE INDEX evidence_exposure_task_idx ON evidence_exposure (project_id, task_id, exposed_at DESC);

-- +goose Down
DROP INDEX IF EXISTS evidence_exposure_task_idx;
DROP INDEX IF EXISTS evidence_exposure_binding_idx;
DROP INDEX IF EXISTS verification_evidence_ruling_evidence_idx;
DROP INDEX IF EXISTS verification_evidence_scope_idx;
DROP INDEX IF EXISTS verification_evidence_run_idx;
DROP INDEX IF EXISTS verification_run_task_idx;
DROP INDEX IF EXISTS verification_run_changeset_idx;
DROP TABLE IF EXISTS evidence_exposure;
DROP TABLE IF EXISTS verification_evidence_ruling;
DROP TABLE IF EXISTS verification_evidence;
DROP TABLE IF EXISTS verification_run;
