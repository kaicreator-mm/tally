-- +goose Up

-- P3-05:outcome_event —— PRD §23 OutcomeLog 的 append-only 事件面。
--
-- PRD §23:v1.4 起立即采集 outcome(accepted / reverted / hotfixed /
-- regression_found / human_correction),因为这些数据事后无法完整补采 ——
-- 一个 revert 发生过就是发生过,没有采集点它就永远消失了。
--
-- 决策锁 14:Outcome 是 append-only 事件捕获,不得建可变 outcome 布尔列当
-- 权威。PRD §23 的五个字段因此不落成 work_item 上的一行可变状态
-- (UPDATE accepted = true),而是每个 outcome 落成一条带 kind 的事件行:
--   - 同一 task 先 accepted 后 reverted,是两行事实,历史完整可重放;
--   - 「当前 outcome」永远是事件流的读侧投影(最近一条、或按 kind 存在性
--     推导),本表不存、也不允许任何可变投影列(AC5:派生结论不得变成
--     可变权威,同 INV-STATE-02 的证据先例)。
--
-- 绑定口径(AC2 / 决策锁 13):PRD §23 写 work_item_id,但锁 13 把 WorkItem
-- 重写推迟到 NEXT,现有 TaskContract 身份(task_id)就是 NOW 的归属坐标;
-- changeset_id 指向被验收/被回退的候选。两列都可空(绑定「可得时」必填):
-- 采集点先于归档完善,事实先落、绑定随可得补齐 —— 回退/热修有时发生在
-- 任务上下文之外(事故热修),拒绝落行等于永久丢失该事实,与 §23 的
-- 动机(事后无法补采)直接冲突。项目归属是经 task_index 的派生 join,
-- 不在事实面重复落 authority(与 0021 measurement 两表同一惯例)。
--
-- 刻意不做的事:
--   - 无 UPDATE 路径、无可变状态列(锁 14):行级写入后只读;语句级触发器
--     复用 0001 的 tally_prevent_modify()(0011 的强化口径:0 行命中的
--     UPDATE/DELETE 语句本身也报错,与 0019/0020 事实面同型,AC3);
--   - event_kind 是封闭枚举(AC1),CHECK 在数据库层拒绝未知 kind ——
--     未知/未匹配的 outcome 在写入时显式失败(fail-safe),不被静默折叠
--     进任何一个已知 kind;
--   - 不建任何 Outcome UI、报表聚合列或「当前状态」投影表(§23:v1.4 只做
--     耐久捕获,无分析平台);
--   - 不加额外索引(migrations/README.md:有真实数据量拿 EXPLAIN 再定)。

-- outcome_event:一条 outcome 事实。
--   event_kind   封闭枚举(AC1,PRD §23 最小集):
--                  accepted           候选被接受(验收闭环终点)
--                  reverted           已接受候选被回退
--                  hotfixed           事故热修(绕过常规候选闭环的修复)
--                  regression_found   已接受候选之后发现回归
--                  human_correction   人工修正(人工改写了 agent 产出)
--   task_id      归属任务(现有 TaskContract 身份,锁 13);可得时必填,可空
--                (上下文外的事件,如事故热修,先落事实)。
--   changeset_id 被验收/回退/热修的候选 ChangeSet;可得时必填,可空。
--   recorded_at  事件写入时间(§23 的 timestamp),写入方显式给出,不由 DB
--                默认 —— 与 0021 事实面同一纪律:缺省时间会把「何时发生的」
--                变成「何时补记的」的猜测。
CREATE TABLE outcome_event (
  id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  task_id      TEXT,
  changeset_id TEXT,
  event_kind   TEXT NOT NULL CHECK (event_kind IN
                 ('accepted','reverted','hotfixed',
                  'regression_found','human_correction')),
  recorded_at  TIMESTAMPTZ NOT NULL
);

CREATE TRIGGER outcome_event_append_only
  BEFORE UPDATE OR DELETE ON outcome_event
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

-- +goose Down
DROP TABLE IF EXISTS outcome_event;
