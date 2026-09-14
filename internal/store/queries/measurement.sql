-- llm_usage / human_effort 的读写合同(P1-13;v1.4 P0-05 演进用量可用性)。
--
-- 两张表都是事实面(migrations/0021),列与 internal/measurement 的事实类型
-- 一一同构:llm_usage ↔ measurement.LLMUsage,human_effort ↔
-- measurement.HumanEffort。本文件只有 INSERT 和 SELECT:没有任何 UPDATE/DELETE
-- (INV-FACT-01,数据库层另有行级触发器兜底),度量事实落库后只读。
--
-- 用量可用性(P0-05,决策锁 16 / PRD §29 item 6-7 / INV-V14-12):
-- migrations/0022 起 llm_usage 的 input_tokens / output_tokens / cost_micros
-- 可空,NULL = 未观测(未知),非 NULL = 已上报的观测值(观测到 0 是合法事实);
-- usage_reported 是写入时必填的显式标志(无默认),0022 的 CHECK 把两者锁成
-- 同一事实。未知用量的调用照样落行 —— 未上报 ≠ 0,不得把「未观测」写成
-- 「观测到 0」的假上报。
--
-- 归属口径(PRD §25.3/§30/§38.4):cost_plane 是写入时落成的事实字段(AC3),
-- 本文件的所有成本聚合都只按存储的 cost_plane 列分组 —— 不存在、也不允许出现
-- 任何 role/use_case→plane 的查询期重算;control_plane_llm_cost_ratio 等报表
-- 由聚合结果在读侧推导。人工计时 active/wall 是两个独立口径(§25 North Star
-- 的分母只认 active),不合并;分钟口径(*_minutes)由读侧换算
-- (measurement.Minutes),库内只存微秒整数,聚合返回微秒和。

-- name: InsertLLMUsage :one
-- 追加一条 LLM/Agent 调用使用/成本事实(§30 llm_run 度量面)。事实面唯一写路径:
-- INSERT 且只 INSERT。cost_plane 必填(0021 CHECK 限定 control|execution,无
-- 默认归属面);task_id 可空(任务外控制面调用);失败/取消调用按实际发生的
-- token/成本照记。
--
-- 用量未知时(P0-05):usage_reported 传 false,input/output/cost 传 NULL ——
-- 调用事实照落,数值留 NULL;0022 的 CHECK 拒绝「未上报却带数值」(假上报)
-- 与「已上报却缺数值」两种不自洽行。
INSERT INTO llm_usage (
  task_id, use_case, cost_plane,
  provider, runtime, model,
  prompt_revision, behavior_revision, context_hash,
  input_tokens, output_tokens, latency_micros, cost_micros, usage_reported,
  status, failure_type, human_intervened, authoring_provenance, recorded_at
) VALUES (
  $1, $2, $3,
  $4, $5, $6,
  $7, $8, $9,
  $10, $11, $12, $13, $14,
  $15, $16, $17, $18, $19
)
RETURNING id;

-- name: ListLLMUsageByTask :many
-- 某任务全部调用记录(旧→新,recorded_at 同刻按 id 定序):逐条审计与
-- by role/runtime/model 维度的读侧透视(§38.4)数据源。usage_reported 随行
-- 返回,逐行的「未知用量」与「观测到 0」在读侧可区分(锁 16)。
SELECT id, task_id, use_case, cost_plane,
       provider, runtime, model,
       prompt_revision, behavior_revision, context_hash,
       input_tokens, output_tokens, latency_micros, cost_micros, usage_reported,
       status, failure_type, human_intervened, authoring_provenance, recorded_at
FROM llm_usage
WHERE task_id = $1
ORDER BY recorded_at, id;

-- name: SumLLMCostByPlane :many
-- 全表按写入时落成的 cost_plane 聚合(每面至多一行,control 在前):
-- control_plane_llm_cost / execution_plane_llm_cost / total 的唯一合法来源。
-- 只读存储列分组,不做任何查询期归属重算(§25.3/§30);比率在读侧相除。
--
-- 用量口径(P0-05,锁 16:未知用量仍是一次调用):
--   - calls_total       = 全部调用,含未知用量的;
--   - calls_reported    = usage_reported 为真的行;
--   - calls_missing_usage = 用量未上报的行 —— 不完整度由它显式可见;
--   - sum 只累计非 NULL(已上报)的数值:未知用量的行对 sum 无贡献,也不会
--     被当作 0 计入「已上报」合计;组内无任何已上报行时 sum 为空,
--     coalesce(..., 0) 把空合计落成 0(空和),此时 calls_reported = 0、
--     calls_missing_usage 指明原因 —— 读侧凭二者区分「0 是空和」与「假 0」。
SELECT cost_plane,
       coalesce(sum(cost_micros), 0)::bigint   AS cost_micros,
       coalesce(sum(input_tokens), 0)::bigint  AS input_tokens,
       coalesce(sum(output_tokens), 0)::bigint AS output_tokens,
       count(*)::bigint                                   AS calls_total,
       count(*) FILTER (WHERE usage_reported)::bigint     AS calls_reported,
       count(*) FILTER (WHERE NOT usage_reported)::bigint AS calls_missing_usage
FROM llm_usage
GROUP BY cost_plane
ORDER BY cost_plane;

-- name: SumLLMCostByPlaneForTask :many
-- 同上,限定单任务:cost per verified task 与按任务的成本分维(§38.4)读侧用。
-- 计数与求和口径与 SumLLMCostByPlane 完全一致(total/reported/missing 分列,
-- 未知用量不进数值合计)。
SELECT cost_plane,
       coalesce(sum(cost_micros), 0)::bigint   AS cost_micros,
       coalesce(sum(input_tokens), 0)::bigint  AS input_tokens,
       coalesce(sum(output_tokens), 0)::bigint AS output_tokens,
       count(*)::bigint                                   AS calls_total,
       count(*) FILTER (WHERE usage_reported)::bigint     AS calls_reported,
       count(*) FILTER (WHERE NOT usage_reported)::bigint AS calls_missing_usage
FROM llm_usage
WHERE task_id = $1
GROUP BY cost_plane
ORDER BY cost_plane;

-- name: InsertHumanEffort :one
-- 追加一条人工控制开销事实(AC4;§25 至少记录条目、§19.4 审核计时)。
-- 事实面唯一写路径:INSERT 且只 INSERT,人工计时行只增不改。kind 由 0021 的
-- CHECK 限定为五个封闭枚举;active/wall 允许其一为 0(只测到其中一个口径),
-- 均不得为负。返回新行 id。
INSERT INTO human_effort (task_id, kind, active_micros, wall_micros, recorded_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: ListHumanEffortByTask :many
-- 某任务全部人工计时行(旧→新,recorded_at 同刻按 id 定序):分段上报的
-- 原始区间全部可见,汇总在读侧做。
SELECT id, task_id, kind, active_micros, wall_micros, recorded_at
FROM human_effort
WHERE task_id = $1
ORDER BY recorded_at, id;

-- name: SumHumanEffortByKindForTask :many
-- 某任务按 kind 聚合的 active/wall 微秒和与样本数(每 kind 至多一行,kind 字典序):
-- contract/verification/diff_review 的 active/wall 分钟、manual_correction、
-- spec_defect_decision 等 *_minutes 指标(§25/§38.2)由此换算,不落新列。
SELECT kind,
       sum(active_micros)::bigint AS active_micros,
       sum(wall_micros)::bigint   AS wall_micros,
       count(*)::bigint           AS samples
FROM human_effort
WHERE task_id = $1
GROUP BY kind
ORDER BY kind;
