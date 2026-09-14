-- +goose Up

-- P1-13:HumanApprovalEvidence / 人工审核计时 / LLM 使用成本三张事实面表。
--
-- PRD §18.1:APPROVED 不是 `UPDATE task SET status`,而是
--   Owner decision → HumanApprovalEvidence → State Engine → APPROVED;
--   即人工决定只以「证据」入库,权威状态由状态引擎另行派生(INV-STATE-02:
--   每一条事实性 lifecycle 状态可追溯到证据)。本迁移因此只建事实表,
--   不建任何任务状态列,也不带任何 UPDATE 路径。
-- PRD §8.10(Waiver):豁免只能覆盖运行期 independence 损失,必须带 provenance
--   (reason code、exposure/acceptance refs、Owner 风险接受);结论区分
--   strict_satisfied / strict_waived,且 strict_waived 不得在任何 UI/报表中
--   表述为「Strict 已满足」。本表存 waived_independence 原始事实,豁免豁掉的
--   是「strict 满足」这一结论 —— strict_waived 永远是可从事实行区分的投影,
--   不会被任何列覆盖或洗白。
-- PRD §19.4/§25/§30/§33.2/§38.4:人工审核计时(active/wall 双口径)与
--   LLM/Agent 使用记录(usage/cost、human review timing 属 PostgreSQL 存储面)
--   按调用/投入发生时写入,之后只读;cost_plane 是写入时落成的历史事实字段,
--   报表按存储值聚合,严禁查询期按 role 重算归属(§25.3/§30)。
--
-- 字段与 internal/measurement(P1-08 事实模型)一一对应:
--   llm_usage    ↔ measurement.LLMUsage(含 CostPlane/UsageStatus 枚举)
--   human_effort ↔ measurement.HumanEffort(含 HumanEffortKind 枚举)
-- 时长统一存微秒整数(latency/active/wall 后缀 _micros),与 measurement 的
-- time.Duration 换算口径固定,避免浮点;分钟口径是读侧换算(measurement.Minutes)。
--
-- 刻意不做的事:
--   - 三张表全部只追加(INV-FACT-01):行级触发器复用 0001 的
--     tally_prevent_modify() 阻止 UPDATE/DELETE,与 0011 的事实面双保险同型;
--   - 不建任何任务状态列、不出任何 UPDATE 查询(AC2):APPROVED 等状态始终
--     由证据派生,human_approval 只提供证据;
--   - human_approval 带 project_id+task_id(lifecycle 证据,遵守 §33.3
--     ghost-project 约束与事实面惯例);measurement 两表只带 task_id、与
--     measurement.LLMUsage/HumanEffort 字段严格同构(任务外控制面调用允许
--     无 task;项目归属是经 task_index 的派生 join,不在事实面重复落 authority);
--   - 不加额外索引(migrations/README.md:有真实数据量拿 EXPLAIN 再定)。

-- human_approval:一条 Owner 决定证据(批准/否决),可选带豁免 provenance。
--   decision     approve|reject(§19.2:Unattributed Changes 非空不得 Approve;
--                否决/不批准一律记 reject,原因必填)
--   reason       决定原因(事实原样记录,必填)
--   decided_by   决定人身份(证据的 recorder 侧写;人工决定是 fact-capable 通道)
--   decided_at   决定时刻(人工给出,不由 DB 默认)
--   changeset_id 被审 ChangeSet(§19:Review 输入是固定 revision 的 ChangeSet;
--                可空:非候选裁决,如合同评审类决定)
--   refs         决定指向的其余坐标(acceptance/verification revision/
--                integration candidate 等,JSON 对象,自包含原样记录)
--   waived_independence / waiver_reason_code / waiver_refs:
--                §8.10 豁免 provenance;豁免时 reason code 必填(CHECK 兜底),
--                waiver_refs 记 compromised acceptances/bindings、exposure refs、
--                coverage gap(JSON 数组)。strict_waived = 本标记为真的行,
--                strict_satisfied = 其余行;二者在行级即可区分,不得混算。
CREATE TABLE human_approval (
  id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id          TEXT NOT NULL,
  task_id             TEXT NOT NULL,
  changeset_id        TEXT,
  decision            TEXT NOT NULL CHECK (decision IN ('approve','reject')),
  reason              TEXT NOT NULL CHECK (reason <> ''),
  decided_by          TEXT NOT NULL CHECK (decided_by <> ''),
  decided_at          TIMESTAMPTZ NOT NULL,
  waived_independence BOOLEAN NOT NULL DEFAULT false,
  waiver_reason_code  TEXT,
  waiver_refs         JSONB NOT NULL DEFAULT '[]',
  refs                JSONB NOT NULL DEFAULT '{}',
  -- 豁免必须带 reason code(§8.10:provenance 至少含 reason code)
  CONSTRAINT human_approval_waiver_needs_reason CHECK (
    waived_independence = false OR (waiver_reason_code IS NOT NULL AND waiver_reason_code <> '')
  )
);
CREATE TRIGGER human_approval_append_only
  BEFORE UPDATE OR DELETE ON human_approval
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

-- llm_usage:一次 LLM/Agent 调用的使用/成本事实(PRD §30 llm_run 度量面)。
-- 列与 measurement.LLMUsage 同名同义;cost_plane 在写入时落成事实字段
-- (AC3,§25.3/§30:禁止查询期 role→plane 重算),无默认归属面,必填。
-- 时长 latency_micros = time.Duration 的微秒整数;cost_micros 为记账货币
-- 1e-6 整数(§25.3 比率一律由它推导)。失败/取消调用按实际发生值照记。
CREATE TABLE llm_usage (
  id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  task_id              TEXT,
  use_case             TEXT NOT NULL CHECK (use_case <> ''),
  cost_plane           TEXT NOT NULL CHECK (cost_plane IN ('control','execution')),
  provider             TEXT NOT NULL DEFAULT '',
  runtime              TEXT NOT NULL DEFAULT '',
  model                TEXT NOT NULL DEFAULT '',
  prompt_revision      TEXT NOT NULL DEFAULT '',
  behavior_revision    TEXT NOT NULL DEFAULT '',
  context_hash         TEXT NOT NULL DEFAULT '',
  input_tokens         BIGINT NOT NULL CHECK (input_tokens >= 0),
  output_tokens        BIGINT NOT NULL CHECK (output_tokens >= 0),
  latency_micros       BIGINT NOT NULL CHECK (latency_micros >= 0),
  cost_micros          BIGINT NOT NULL CHECK (cost_micros >= 0),
  status               TEXT NOT NULL CHECK (status IN ('succeeded','failed','cancelled')),
  failure_type         TEXT NOT NULL DEFAULT '',
  human_intervened     BOOLEAN NOT NULL DEFAULT false,
  authoring_provenance TEXT NOT NULL DEFAULT '',
  recorded_at          TIMESTAMPTZ NOT NULL
);
CREATE TRIGGER llm_usage_append_only
  BEFORE UPDATE OR DELETE ON llm_usage
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

-- human_effort:一条人工控制开销事实(PRD §25 至少记录条目、§19.4 审核计时)。
-- kind 是封闭枚举(与 measurement.HumanEffortKind 一致):
--   contract_review / verification_review / diff_review /
--   manual_correction / spec_defect_decision。
-- active(人实际投入)与 wall(端到端等待)是两个口径,不合并(§25 North Star
-- 的分母只认 active);不强制 wall >= active,两口径可来自不同测量窗。
CREATE TABLE human_effort (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  task_id       TEXT NOT NULL CHECK (task_id <> ''),
  kind          TEXT NOT NULL CHECK (kind IN
                  ('contract_review','verification_review','diff_review',
                   'manual_correction','spec_defect_decision')),
  active_micros BIGINT NOT NULL CHECK (active_micros >= 0),
  wall_micros   BIGINT NOT NULL CHECK (wall_micros >= 0),
  recorded_at   TIMESTAMPTZ NOT NULL
);
CREATE TRIGGER human_effort_append_only
  BEFORE UPDATE OR DELETE ON human_effort
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

-- +goose Down
DROP TABLE IF EXISTS human_effort;
DROP TABLE IF EXISTS llm_usage;
DROP TABLE IF EXISTS human_approval;
