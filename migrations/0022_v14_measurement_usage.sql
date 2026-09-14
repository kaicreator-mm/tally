-- +goose Up

-- P0-05:llm_usage 显式表示「用量可用性」(决策锁 16 / PRD §29 item 6-7 /
-- INV-V14-12)。
--
-- 0021 的合同把 input_tokens / output_tokens / cost_micros 设为 NOT NULL,
-- 于是「provider 未上报用量的调用」无法落成事实行:要么被写入方静默丢弃
-- (当前 internal/measurement 的行为:Tokens 未上报就不落行),要么被写成
-- 全 0 —— 后者正是把「未观测」伪装成「观测到 0」的假上报,锁 16 明令禁止。
-- 两种结果都丢失了「这次调用确实发生过」的事实。
--
-- 本迁移是 brownfield 有界改动(retain → repair),不重写 0021:
--   1. token/cost 三列 DROP NOT NULL:NULL = 未观测(未知),非 NULL = 已上报
--      的观测值(观测到 0 是合法事实,与 NULL 语义不可互换);
--   2. 新增显式标志列 usage_reported(写入时必填,无默认):本调用的用量
--      是否被 provider/runtime 上报;
--   3. CHECK 把 1 与 2 锁成同一事实:上报 ⇒ 三列全部非 NULL;未上报 ⇒
--      三列全部 NULL。双向兜底,任何把未知写成假 0(或已上报却缺数值)的
--      写入在数据库层被拒。
--
-- 存量行语义:0021 时代只有真实上报过数值的调用才会落行(写入方跳过未上报
-- 调用),因此回填 usage_reported = true 是对既有事实的如实标注,不是猜测。
-- 回填用 ADD COLUMN ... DEFAULT true 完成;随后立即 DROP DEFAULT —— 之后
-- 的写入必须显式声明可用性,一个缺省的 true 会把未来的未知调用悄悄伪装成
-- 已上报(与 0021 cost_plane「无默认归属面」同一纪律)。
--
-- latency_micros / status / failure_type 维持 NOT NULL 不动:Tally 自己观测
-- 调用墙钟与终态,与 provider 是否上报用量无关;未知用量的调用照样有
-- latency 与终态。行级 append-only 触发器(0021)不受 DDL 影响,事实行
-- 落库后仍只读(INV-FACT-01)。

ALTER TABLE llm_usage
  ALTER COLUMN input_tokens  DROP NOT NULL,
  ALTER COLUMN output_tokens DROP NOT NULL,
  ALTER COLUMN cost_micros   DROP NOT NULL;

ALTER TABLE llm_usage
  ADD COLUMN usage_reported BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE llm_usage
  ALTER COLUMN usage_reported DROP DEFAULT;

ALTER TABLE llm_usage
  ADD CONSTRAINT llm_usage_usage_reported_consistent CHECK (
    (
      usage_reported
      AND input_tokens  IS NOT NULL
      AND output_tokens IS NOT NULL
      AND cost_micros   IS NOT NULL
    )
    OR (
      NOT usage_reported
      AND input_tokens  IS NULL
      AND output_tokens IS NULL
      AND cost_micros   IS NULL
    )
  );

-- +goose Down

-- 回退到 0021 合同。若已存在未知用量的行,恢复 NOT NULL 会被数据库拒绝
-- (NULL 值无真值可补):Down 只做合同层回退,不为未知事实编造数值。
ALTER TABLE llm_usage
  DROP CONSTRAINT llm_usage_usage_reported_consistent;
ALTER TABLE llm_usage
  ALTER COLUMN input_tokens  SET NOT NULL,
  ALTER COLUMN output_tokens SET NOT NULL,
  ALTER COLUMN cost_micros   SET NOT NULL;
ALTER TABLE llm_usage
  DROP COLUMN usage_reported;
