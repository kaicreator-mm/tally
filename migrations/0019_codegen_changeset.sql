-- +goose Up

-- P1-11:CodegenRun / ChangeSet 持久化 SQL 契约
-- (PRD §15 CodegenRun、§6.4.2 ChangeSet、§11.2.1 escalation、INV-CHANGE-02)。
--
-- 三张表,三种持久化语义,刻意不混用:
--
--   codegen_run                过程记录(PRD §15.1):一次正式 Agent 执行是一等对象。
--                              Run 有生命周期状态迁移(§17 「CodegenRun 状态迁移」、
--                              §11.2.1 「status: escalated → terminate」):行在运行开始时
--                              以 status='running' 写入,由 workflow kernel 推移到终态。
--                              因此本表不设 append-only 触发器,与事实面(0001/0011)相反;
--                              单值事实语义(runtime/model/behavior 一 Run 一值,§11.2.1)
--                              由列不可空 + escalated_from 自引用保证:升级不得在同一 Run
--                              内替换 runtime/model,只能终止旧 Run、新 Run 记 escalated_from。
--
--   changeset                  不可变候选快照(INV-CHANGE-02:验证/评审/集成的固定单元)。
--                              一个 Run may produce 0..1 个候选快照(决策锁 8),
--                              producer_run_id UNIQUE 在数据库层强制;没有固定 candidate
--                              的 Run(crash/预算耗尽)不产生行(PRD §11.2.1/§15.3)。
--                              快照本体只 INSERT:UPDATE/DELETE 一律触发器拦截 ——
--                              后续重做产生新快照;UI 要的 superseded_by 是投影,
--                              快照本体不带也不得带可变取代字段(PRD §15.3,AC2)。
--
--   changeset_superseded_fact  取代关系 append-only 事实(PRD §15.3):
--                              old/new/reason 只追加,永不修改,触发器拦截 UPDATE/DELETE(AC3)。
--                              reason 取值固定为 escalation | rework | spec_defect。

-- ① codegen_run:一次正式 Agent 执行的过程记录(PRD §15.1 全字段)。
-- 身份是代理主键 id;可追溯坐标(PRD §15.2:同一 Task 为什么某次成功/某次失败,
-- 能定位 code baseline、TaskContract、VerificationContract、ContextPack、
-- Runtime/Model/Behavior)全部非空,在运行开始时即已知。
CREATE TABLE codegen_run (
  id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id          TEXT NOT NULL,
  task_id             TEXT NOT NULL,               -- TaskContract 身份(Git 持有意图,这里只存引用,DECISION_LOCKS 9)
  role                TEXT NOT NULL,               -- 本 Run 承担的 Agent Role
  runtime             TEXT NOT NULL,               -- Runtime Adapter 名
  runtime_version     TEXT NOT NULL,
  model               TEXT NOT NULL,
  behavior_revision   TEXT NOT NULL,               -- BehaviorManifest/行为修订(PRD §15.2 可归因)
  cost_plane          TEXT NOT NULL CHECK (cost_plane IN ('control','execution')),
                      -- 成本归属面,调用发生时写入,不做查询期推导(PRD §30;
                      -- 取值与 internal/measurement.CostPlane 一致)
  base_commit         TEXT NOT NULL,               -- code baseline(PRD §15.2)
  task_revision       TEXT NOT NULL,               -- 执行所对 TaskContract 修订哈希
  verification_revision TEXT NOT NULL,             -- 执行所对 VerificationContract 修订哈希(P1 standard 冻结在前)
  context_pack_hash   TEXT NOT NULL,               -- 本 Run 消费的 ContextPack 哈希(§12)
  workspace_id        TEXT NOT NULL,               -- 一次性 Git worktree(P0-05/P1-35)
  agent_session_id    TEXT,                        -- Runtime 会话身份;会话未及建立即崩溃时为 NULL
  started_at          TIMESTAMPTZ NOT NULL,
  finished_at         TIMESTAMPTZ,
  status              TEXT NOT NULL CHECK (status IN ('running','completed','failed','escalated','cancelled')),
                      -- 终态迁移由 §11.2.1/§17 语义驱动;Task 级 NEEDS_REVIEW 是派生
                      -- 生命周期状态(INV-STATE),不是 Run 状态,不进本列取值
  iterations          INTEGER NOT NULL DEFAULT 0 CHECK (iterations >= 0),
  escalated_from      BIGINT REFERENCES codegen_run(id),
                      -- 升级而新开的 Run 指向被终止的旧 Run(PRD §11.2.1);非升级 Run 为 NULL
  input_tokens        BIGINT CHECK (input_tokens IS NULL OR input_tokens >= 0),   -- NULL=尚未上报
  output_tokens       BIGINT CHECK (output_tokens IS NULL OR output_tokens >= 0),
  cost_micros         BIGINT CHECK (cost_micros IS NULL OR cost_micros >= 0),
                      -- 记账货币 1e-6(micro),与 internal/measurement 的 CostMicros 同单位
  result_patch_hash   TEXT,                        -- 产出候选 patch 身份;没有固定 candidate 的 Run 为 NULL(PRD §11.2.1)
  agent_claim         TEXT,                        -- Agent 的 Claim 原样记录;Claim 永非 Project Fact(INV-AUTH-01/02)
  validation_result   TEXT,                        -- 确定性 validation 的记录(写入方编码,不在此定形)
  failure_reason      TEXT,                        -- 终态为 failed/escalated/cancelled 时的原因
  runtime_capability_snapshot JSONB NOT NULL,      -- 运行时声明能力快照(§10.3,自包含冗余,同 version_combo 惯例)
  conformance_evidence_ref  TEXT,                  -- Runtime conformance evidence 引用(§10.4);NULL=本 Run 未绑定
  -- 状态与结束时刻自洽:running 必未结束,终态必已结束
  CONSTRAINT codegen_run_status_finished CHECK (
    (status = 'running' AND finished_at IS NULL)
    OR (status <> 'running' AND finished_at IS NOT NULL)
  )
);

-- Task 时间线(CodegenRun timeline,PRD §29 视图族)是 P1 热路径。
CREATE INDEX codegen_run_task_timeline_idx ON codegen_run (project_id, task_id, started_at);

-- ② changeset:不可变候选快照(PRD §15.3 / §6.4.2,INV-CHANGE-02)。
-- 身份 id 由内容推导(project+base+candidate+patch+producer run,sha256:<hex>,
-- 与 internal/changeset.deriveID 一致):可重算、确定,同一 Run 的同一候选同 ID。
CREATE TABLE changeset (
  id                  TEXT PRIMARY KEY,            -- "sha256:<hex>",internal/changeset 推导,写入方先算好
  project_id          TEXT NOT NULL,
  base_commit         TEXT NOT NULL,               -- 固定的 base revision(PRD §6.4.2)
  candidate_commit    TEXT,                        -- 固定的 candidate revision;纯 patch 身份时为 NULL
  patch_hash          TEXT,                        -- 改动内容推导的 patch 身份("sha256:<hex>");纯 commit 候选时为 NULL
  changed_files       JSONB NOT NULL,              -- 改动文件/hunks 全量快照(自包含,patch_hash 覆盖其内容;
                                                   -- 编码形状由写入方适配器按 internal/changeset.ChangedFile 定)
  entities            JSONB NOT NULL DEFAULT '[]'::jsonb,
                                                   -- 受影响 SoftwareEntity(PRD §6.4.2「可推导时」,可为空;对象本身 P2)
  producer_run_id     BIGINT NOT NULL UNIQUE REFERENCES codegen_run(id),
                                                   -- 产生本快照的 Run;UNIQUE = 一 Run 0..1 个候选快照(决策锁 8)
  created_at          TIMESTAMPTZ NOT NULL,        -- 快照固定时刻(快照方 UTC,PRD §15.3)
  -- 候选身份:candidate revision 与 patch 身份至少其一(§6.4.2「candidate_commit 或 patch hash」)
  CONSTRAINT changeset_candidate_identity CHECK (candidate_commit IS NOT NULL OR patch_hash IS NOT NULL)
);

CREATE TRIGGER changeset_append_only
  BEFORE UPDATE OR DELETE ON changeset
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

-- ③ changeset_superseded_fact:取代关系 append-only 事实(PRD §15.3,AC3)。
-- old/new 引用 changeset(id)—— id 本身就是内容推导哈希;列名随被引列(id)。
CREATE TABLE changeset_superseded_fact (
  id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  old_changeset_id    TEXT NOT NULL REFERENCES changeset(id),
  new_changeset_id    TEXT NOT NULL REFERENCES changeset(id),
  reason              TEXT NOT NULL CHECK (reason IN ('escalation','rework','spec_defect')),
  superseded_at       TIMESTAMPTZ NOT NULL,        -- 事实发生时刻(事实方 UTC)
  -- old 与 new 必不相同(与 internal/changeset.NewSupersededFact 同一约束)
  CONSTRAINT changeset_superseded_distinct CHECK (old_changeset_id <> new_changeset_id),
  -- 同一取代事实重复追加(重试)幂等去重;不同时刻的取代是不同事实,各自成行
  CONSTRAINT changeset_superseded_fact_unique UNIQUE (old_changeset_id, new_changeset_id, reason, superseded_at)
);

CREATE TRIGGER changeset_superseded_fact_append_only
  BEFORE UPDATE OR DELETE ON changeset_superseded_fact
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

-- +goose Down
DROP TABLE IF EXISTS changeset_superseded_fact;
DROP TABLE IF EXISTS changeset;
DROP INDEX IF EXISTS codegen_run_task_timeline_idx;
DROP TABLE IF EXISTS codegen_run;
