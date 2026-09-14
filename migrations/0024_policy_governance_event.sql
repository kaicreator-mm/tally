-- +goose Up

-- P3-02:policy_governance_event —— 策略治理的 append-only 事件面。
--
-- BI-01:治理生命周期元数据(status/evidence_refs/approved_by/supersedes/
-- scope)住在本表代表的治理事实侧,绝不进 engineeringpolicy.Policy 的
-- YAML/struct(DL-08);运行时生效的永远是 pinned Git 快照里
-- .tally/policies/*.yaml 的确切内容(DL-09)。
--
-- 刻意不做的事(AC:提案历史可重建;不引入可变运行时策略权威):
--   - 不建任何「当前 ACTIVE 策略」表/布尔列/投影列:激活只是本表里一条
--     action_kind='activate' 的事件行,「runtime consumes ACTIVE」的语义由
--     P3-07 把正典候选 YAML 经 gitwriter 提交到 .tally/policies/ 实现 ——
--     本表对运行时不可见,运行时激活永不从 DB 标志推导(要求 2);
--   - 历史 = 事件流本身:每个提案的最终治理状态由事件按序重放重建
--     (policygovernance.Reconstruct),本表不存「当前状态」可变列;
--   - 无 UPDATE 路径:语句级触发器复用 0001 的 tally_prevent_modify()
--     (0011/0019/0020/0023 同型:0 行命中的 UPDATE/DELETE 语句本身也报错);
--   - action_kind 是封闭枚举,CHECK 在数据库层拒绝未知 kind —— 未知治理
--     动作在写入时显式失败(fail-safe),不被静默折叠;
--   - candidate_yaml 落原始字节文本(所见即所批:P3-07 批准提交的就是这份
--     内容),candidate_rev 落 engineeringpolicy 内容修订哈希(内容派生,
--     可与 Git 面核对);二者都不构成运行时权威,只是治理记录;
--   - 不加额外索引(migrations/README.md:有真实数据量拿 EXPLAIN 再定)。

-- policy_governance_event:一条治理事实。
--   proposal_id     治理身份(域层校验:非空、无空白、≤128)。
--   action_kind      封闭枚举:
--                     draft     草稿落记(候选内容出生即固定)
--                     submit    提交评审(证据 ≥1)
--                     activate  批准并激活(approved_by 必填;Git 提交由 P3-07)
--                     reject    评审否决(终态)
--                     supersede 被后续提案取代(终态;append-only 指针)
--   candidate_rev   候选策略内容修订(engineeringpolicy.RevisionHash)。
--   candidate_yaml  候选策略原始 YAML 文本(所见即所批;非运行时权威)。
--   evidence_refs   证据引用串数组(outcome:<id> / changeset:<id> /
--                   verification:<rev> / task:<id>;封闭 kind 词表)。
--   proposed_by     提案人(域层校验非空)。
--   approved_by     批准人(仅 activate 事件必填)。
--   supersedes      被取代的前任提案 ID(可空)。
--   scope           适用范围说明(可空;治理元数据)。
--   occurred_at     治理动作发生时刻,写入方显式给出,不由 DB 默认 ——
--                   与 0021/0023 事实面同一纪律:缺省时间会把「何时发生的」
--                   变成「何时补记的」的猜测。
CREATE TABLE policy_governance_event (
  id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  proposal_id    TEXT NOT NULL,
  action_kind     TEXT NOT NULL CHECK (action_kind IN
                   ('draft','submit','activate','reject','supersede')),
  candidate_rev  TEXT NOT NULL,
  candidate_yaml TEXT NOT NULL,
  evidence_refs  TEXT[] NOT NULL DEFAULT '{}',
  proposed_by    TEXT NOT NULL,
  approved_by    TEXT,
  supersedes     TEXT,
  scope          TEXT NOT NULL DEFAULT '',
  occurred_at    TIMESTAMPTZ NOT NULL
);

CREATE TRIGGER policy_governance_event_append_only
  BEFORE UPDATE OR DELETE ON policy_governance_event
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

-- +goose Down
DROP TABLE IF EXISTS policy_governance_event;
