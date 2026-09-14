// v14_attention.go 是 v1.4 P3-09 的最小 Attention 投影页(PRD §26.4:只显示
// 真正需要人的事项)。
//
// 投影纪律(源事实:Attention is a projection, not a business object):
//   - Attention 是纯派生投影,不是业务对象:不落库、无 setter、无状态机
//     (PRD §22 AttentionQueue 由底层事实稳定重算;INV-STATE-01);
//   - 只包含合法阻断来源(AC4),词表封闭:taskstate 派生的异常态事实
//     (needs_review / blocked / 终态失败 Run / 终态取消)与在账暴露事实
//     (exposure_compromise 的事实面形态,P2-06 append-only 账面)。此外只有
//     「投影本身失败」一种诚实条目 —— 读取/派生失败的任务绝不静默缺席;
//   - 闭环升级原因词(repair_exhausted / router_unavailable 等)目前是闭环
//     终态日志(cmd/worker v14.go),还没有可读取的事实面投影;其事实面形态
//     (终态失败 Run + 失败原因)照常呈现。本页不发明第二种来源;
//   - waiver 裁决请求没有独立事实来源:其事实面形态是 NEEDS_REVIEW 异常态与
//     覆盖门槛缺口,不另造来源(与 v1.3 UI 投影同一边界);
//   - Dismiss/snooze 未接线(AC5):接入时必须落在独立持久层,生命周期状态
//     不复制、不从本页写 —— 本页没有任何写路由。
//
// 路由经 v14_change.go 的 registerV14ChangeAttentionRoutes 挂载(P3-09 最小
// 注册接入点)。
package handlers

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"tally/internal/artifacts"
	"tally/internal/core"
	"tally/internal/runstore"
	"tally/internal/taskstate"
)

// V14AttentionKind 是 Attention 条目类型的封闭词表(AC4:只有合法阻断来源;
// 未知来源编译期就不存在)。字符串值即页面呈现的机器词面。
type V14AttentionKind string

const (
	// V14AttnNeedsReview:存在未解除的 review 请求(taskstate.NEEDS_REVIEW)。
	V14AttnNeedsReview V14AttentionKind = "needs_review"
	// V14AttnBlocked:存在未解除的显式阻塞(taskstate.BLOCKED)。
	V14AttnBlocked V14AttentionKind = "blocked"
	// V14AttnFailedRun:最近一次实现 Run 终态失败(taskstate.FAILED;闭环升级
	// 原因词的事实面形态 —— 修复耗尽等升级最终都以这里的事实可读)。
	V14AttnFailedRun V14AttentionKind = "failed_run"
	// V14AttnCancelled:取消是终态事实(taskstate.CANCELLED)。
	V14AttnCancelled V14AttentionKind = "cancelled"
	// V14AttnExposure:在账暴露事实(closedloop exposure_compromise 升级的
	// append-only 账面形态;祖先锚定的暴露同样在账,P0-07)。
	V14AttnExposure V14AttentionKind = "exposure_compromise"
	// V14AttnProjectionError:该任务的 Attention 投影本身失败(事实面读取或
	// 状态派生)。诚实条目:投影失败的任务不得静默缺席。
	V14AttnProjectionError V14AttentionKind = "projection_error"
)

// rank 是确定性的条目排序位(同一任务内按事实严重度;投影失败最后)。
func (k V14AttentionKind) rank() int {
	switch k {
	case V14AttnNeedsReview:
		return 0
	case V14AttnBlocked:
		return 1
	case V14AttnFailedRun:
		return 2
	case V14AttnCancelled:
		return 3
	case V14AttnExposure:
		return 4
	default:
		return 5
	}
}

// label 是类型的固定人读标签(封闭 switch,不做字符串拼接推断)。
func (k V14AttentionKind) label() string {
	switch k {
	case V14AttnNeedsReview:
		return "需要人工评审"
	case V14AttnBlocked:
		return "被显式阻塞"
	case V14AttnFailedRun:
		return "实现 Run 终态失败"
	case V14AttnCancelled:
		return "已取消(终态)"
	case V14AttnExposure:
		return "暴露事实在账"
	default:
		return "投影失败"
	}
}

// action 是该类条目的固定目标动作(封闭 switch,不做字符串拼接推断):每条
// 评审项必须同时携带原因(Detail)、依据(Refs)与目标动作(P2-04 AC)。动作
// 文本只复述本页既有边界声明中的人本裁决项,不发明新语义、不带任何写路由。
func (k V14AttentionKind) action() string {
	switch k {
	case V14AttnNeedsReview:
		return "人工裁决该评审请求;裁决后新事实会改变派生结论。"
	case V14AttnBlocked:
		return "人工解除阻塞或补充阻塞依据;解除后新事实会改变派生结论。"
	case V14AttnFailedRun:
		return "人工裁决后续:修复重试 / 换基线 / 放弃(不自动重试)。"
	case V14AttnCancelled:
		return "如需继续,发起新任务(取消是终态,没有解除路径)。"
	case V14AttnExposure:
		return "人工裁决 strict 主张:waiver / 换基线 / 放弃。"
	default:
		return "先修复事实面/派生错误,再评估是否需要人。"
	}
}

// V14AttentionItem 是一条需要 owner 关注的事项:只读投影,携带依据引用
// (生命周期事实 id / 暴露事实 id),可追溯到具体事实。
type V14AttentionItem struct {
	TaskID string
	Kind   V14AttentionKind
	// Label 是类型的固定人读标签。
	Label string
	// Detail 是从事实投影的确定性说明(含失败原因等观测值)。
	Detail string
	// Action 是该类条目的固定目标动作(P2-04 AC:每条评审项必须携带)。
	Action string
	// Refs 是依据引用(fact:<id> / exposure:<id>),不是结论。
	Refs []string
	// At 是最近一条相关事实的观测时刻(无行级时刻的派生条目为空)。
	At string
}

// V14AttentionCount 是聚合小计的一行(类型 × 条数,固定排序位次序)。
type V14AttentionCount struct {
	Kind  V14AttentionKind
	Label string
	Count int
}

// V14AttentionPage 是 Attention 投影页数据。
type V14AttentionPage struct {
	Page
	Unavailable bool // 数据源未接入
	LoadErr     string
	ProjectID   string
	// TaskCount 是本次扫描的 standard TaskContract 数(Attention 的扫描范围)。
	TaskCount int
	Items     []V14AttentionItem
	// Summary 是聚合小计:按类型排序位次序列出非零类别的条数(P2-04:异常 /
	// 高危 / 待裁决的聚合面)。纯投影算术,不落库。
	Summary []V14AttentionCount
}

// V14Attention 渲染 Attention 投影(PRD §26.4):从阻断事实派生,只读无写。
func (s *Server) V14Attention(w http.ResponseWriter, r *http.Request) {
	page := V14AttentionPage{Page: s.page(r, "Attention · v1.4", "v14-attention")}
	env, ok := s.v13env()
	if !ok {
		page.Unavailable = true
		v14CARender(w, http.StatusOK, "v14_attention", page)
		return
	}
	g, err := openV13Git(env.repoPath)
	if err != nil {
		page.LoadErr = err.Error()
		v14CARender(w, http.StatusOK, "v14_attention", page)
		return
	}
	snap, err := artifacts.Load(env.repoPath, g.headHash)
	if err != nil {
		page.LoadErr = err.Error()
		v14CARender(w, http.StatusOK, "v14_attention", page)
		return
	}
	env.project = core.ProjectID(snap.ProjectID)
	page.ProjectID = snap.ProjectID

	ctx := r.Context()
	for _, ta := range snap.Tasks {
		if ta.Contract.Profile != "standard" {
			continue // 闭环路径只执行 standard TaskContract(cmd/worker 同一判据)
		}
		page.TaskCount++
		page.Items = append(page.Items, s.v14AttentionForTask(ctx, env, g, snap, ta)...)
	}
	// 输出确定性:任务 id 升序,任务内按类型排序位(稳定排序保持条目次序)。
	sort.SliceStable(page.Items, func(i, j int) bool {
		if page.Items[i].TaskID != page.Items[j].TaskID {
			return page.Items[i].TaskID < page.Items[j].TaskID
		}
		return page.Items[i].Kind.rank() < page.Items[j].Kind.rank()
	})
	// 聚合小计:按类型排序位次序统计非零类别(P2-04;纯派生算术)。
	counts := map[V14AttentionKind]int{}
	for _, it := range page.Items {
		counts[it.Kind]++
	}
	for _, k := range []V14AttentionKind{
		V14AttnNeedsReview, V14AttnBlocked, V14AttnFailedRun,
		V14AttnCancelled, V14AttnExposure, V14AttnProjectionError,
	} {
		if counts[k] > 0 {
			page.Summary = append(page.Summary, V14AttentionCount{Kind: k, Label: k.label(), Count: counts[k]})
		}
	}
	v14CARender(w, http.StatusOK, "v14_attention", page)
}

// v14AttentionForTask 派生一个 standard 任务的 Attention 条目。只消费合法阻断
// 来源:派生异常态、在账暴露事实;读取/派生失败产出 projection_error 条目,
// 绝不静默缺席。主线推进中的任务产出零条目(Attention 不是任务列表)。
func (s *Server) v14AttentionForTask(ctx context.Context, env v13Env, g *v13Git, snap *artifacts.Snapshot, ta artifacts.TaskArtifact) []V14AttentionItem {
	taskID := ta.Contract.ID

	runs, err := env.runs.CodegenRunsForTask(ctx, env.project, taskID)
	if err != nil {
		return v14ProjectionError(taskID, "实现 Run 读取失败: "+err.Error())
	}
	css, err := env.runs.ChangeSetsForTask(ctx, env.project, taskID)
	if err != nil {
		return v14ProjectionError(taskID, "候选读取失败: "+err.Error())
	}
	vruns, err := env.runs.VerificationRunsForTask(ctx, env.project, taskID)
	if err != nil {
		return v14ProjectionError(taskID, "验证 Run 读取失败: "+err.Error())
	}
	approvals, err := env.approvals.ApprovalsForTask(ctx, env.project, taskID)
	if err != nil {
		return v14ProjectionError(taskID, "审批证据读取失败: "+err.Error())
	}
	exposures, err := env.runs.ExposuresForTask(ctx, env.project, taskID)
	if err != nil {
		return v14ProjectionError(taskID, "暴露事实读取失败: "+err.Error())
	}

	va := v13FrozenFor(snap, taskID)
	res, err := taskstate.Derive(taskstate.Snapshot{
		Contract:  v13FrozenContract(va),
		Facts:     v13FactsFor(&ta, va, g.headTime, runs, css, vruns),
		Approvals: v13ApprovalEvidence(approvals),
	})
	if err != nil {
		return v14ProjectionError(taskID, "状态派生失败: "+err.Error())
	}

	var items []V14AttentionItem
	add := func(kind V14AttentionKind, detail string, refs []string, at string) {
		items = append(items, V14AttentionItem{
			TaskID: taskID, Kind: kind, Label: kind.label(), Detail: detail,
			Action: kind.action(), Refs: refs, At: at,
		})
	}

	// 派生异常态 = 合法阻断来源的第一组(异常由事实推导,不是写上去的标志位)。
	switch res.State {
	case taskstate.StateNeedsReview:
		add(V14AttnNeedsReview, "存在未解除的 review 请求(预算耗尽 / SpecDefect 提案等);裁决后新事实会改变派生结论。", res.Basis, "")
	case taskstate.StateBlocked:
		add(V14AttnBlocked, "存在未解除的显式阻塞;解除后新事实会改变派生结论。", res.Basis, "")
	case taskstate.StateFailed:
		reason, at := v14LatestFailedRun(runs)
		add(V14AttnFailedRun, "最近一次实现 Run 终态失败:"+reason+"(闭环升级原因词的事实面形态;修复/换基线由人裁决)。", res.Basis, at)
	case taskstate.StateCancelled:
		add(V14AttnCancelled, "取消是终态事实,没有解除路径;继续需要新的事实。", res.Basis, "")
	}

	// 在账暴露事实 = exposure_compromise 升级的 append-only 账面形态:strict
	// 信息独立主张被封顶,需要人裁决(waiver / 换基线 / 放弃),绝不静默。
	if len(exposures) > 0 {
		refs := make([]string, 0, len(exposures))
		latest := exposures[0].ExposedAt()
		for _, e := range exposures {
			refs = append(refs, "exposure:"+string(e.ID()))
			if e.ExposedAt().After(latest) {
				latest = e.ExposedAt()
			}
		}
		add(V14AttnExposure,
			strconv.Itoa(len(exposures))+" 条暴露事实在账:held-out 材料进入过实现侧可见通道;谱系内保证只降不升(P0-07),strict 主张需人工裁决。",
			refs, v13FormatTime(latest))
	}
	return items
}

// v14LatestFailedRun 取最近一次终态失败 Run 的失败原因与观测时刻(行未结束
// 时回退开始时刻;没有失败原因时如实说明)。失败原因是观测值,原样呈现,
// 不二次解释。
func v14LatestFailedRun(runs []runstore.Run) (reason string, at string) {
	reason = "未记录失败原因"
	var latest time.Time
	have := false
	for _, r := range runs {
		if r.Status != runstore.RunFailed {
			continue
		}
		cur := r.FinishedAt
		if cur.IsZero() {
			cur = r.StartedAt
		}
		if have && !cur.After(latest) {
			continue
		}
		reason = "未记录失败原因"
		if r.FailureReason != "" {
			reason = r.FailureReason
		}
		latest = cur
		have = true
		at = v13FormatTime(cur)
	}
	return reason, at
}

// v14ProjectionError 是「投影本身失败」的诚实条目:该任务不得静默缺席,
// 失败原因原样呈现。
func v14ProjectionError(taskID, detail string) []V14AttentionItem {
	return []V14AttentionItem{{
		TaskID: taskID,
		Kind:   V14AttnProjectionError,
		Label:  V14AttnProjectionError.label(),
		Detail: detail + "(Attention 对该任务不可信:先修复事实面/派生错误,再评估是否需要人。)",
		Action: V14AttnProjectionError.action(),
		Refs:   nil,
		At:     "",
	}}
}
