// v13_runs.go 是 v1.3/v1.4 界面的 Runs 页(P2-03,PRD §6.1 Run、§29 Runs、
// §33 Phase 3「Human-Centered UI Projection:Runs,不改状态机」)。
//
// 页面语义(任务 AC:Run 状态来自既有 workflow/runtime 事实,不造第二套运行状态):
//   - 行是 codegen_run 过程记录(runstore.Run)的原样只读投影:status 是行上的
//     run-control 事实(running/completed/failed/escalated/cancelled,
//     migrations/0019 CHECK),本页原样呈现,不发明新状态、不迁移任何 Run;
//   - 期望坐标(runtime/model/profile,§15.2 开跑写定)与 runtime 现场自报的
//     实际身份(observed_identity,Decision Lock 15)分开呈现,互不冒充;
//   - 清单经任务清单展开:任务枚举来自 HEAD 制品快照(Git 是 desired 制品唯一
//     来源),逐任务走既有 CodegenRunsForTask 读路径(与工作列表/Attention 同一
//     读取纪律;sqlc 查询集只有按任务的运行列表,本页不为此新增存储查询);
//     不在 HEAD 快照里的历史任务,其 Run 不在此页(模板显式说明,不静默);
//   - 只读投影(DL-19):没有任何状态字段/写入控件;任务派生状态归工作页,
//     本页只陈述实际发生了什么。
package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"

	"tally/internal/artifacts"
	"tally/internal/core"
	"tally/internal/runstore"
)

// v13ObservedIdentityKey 是 runstore 写入实际身份事实的保留顶层键
// (internal/runstore 的 observedIdentityKey;此处只读回放,不改写编码)。
const v13ObservedIdentityKey = "observed_identity"

// v13ObservedIdentity 是 observed_identity 键的只读投影形状(与 runstore 的
// 写入编码对应;读取容错:解析失败给显式说明,不冒充未观察)。
type v13ObservedIdentity struct {
	RuntimeName    string `json:"runtime_name"`
	RuntimeVersion string `json:"runtime_version"`
	Model          string `json:"model"`
}

// V13RunRow 是 Runs 页的一行:一条 codegen_run 事实的展示投影。
type V13RunRow struct {
	ID    string
	Task  string
	Title string // 任务标题(HEAD 快照内该 TaskContract 的;空 = 无标题)

	// Status 是 codegen_run 行上的 run-control 事实原样呈现(不二次解释)。
	Status      string
	StatusClass string // 展示色映射(st-ok/st-warn/st-bad/st-off);文本保持原样

	// 期望坐标(§15.2 开跑写定;升级不得在 Run 内替换;实际身份另行观察)。
	Runtime string
	Model   string
	// Observed 是 runtime 现场自报的实际身份摘要;空 = 未观察(合法,
	// 零值降级,不用期望值补齐 —— Decision Lock 15)。
	Observed string
	// ObservedNote 是实际身份快照存在但无法解析时的显式说明(不为空时
	// Observed 必为空):快照坏了要说出来,不静默当作未观察。
	ObservedNote string

	EscalatedFrom string // 非 0 = 升级链来源 Run 行 id(升级即换 Run,§11.2.1)

	Iterations int32
	Tokens     string // 「in N · out M」;空 = 尚未上报(nil 语义原样保留)
	Started    string
	Finished   string // 「—」= 未结束(running)

	// FailureReason 是终态失败/升级/取消原因词的事实面形态,原样呈现不解释
	// (runtime_error / wall_time_budget_exhausted / escalated_to:<profile> …)。
	FailureReason string

	// CandidateID 非空 = 该 Run 固定了候选(ResultPatchHash 即 ChangeSet 身份),
	// 行内给出候选页链接;失败/升级/取消的 Run 恒空(§11.2.1)。
	CandidateID string

	// id/startedAt 是排序用的原始值(不进模板;展示用上面的字符串形态)。
	id        int64
	startedAt time.Time
}

// V13TaskRunErr 是一个任务的实现 Run 读取失败的显式行:该任务不得静默缺席
// (与 Attention 的 projection_error 同一诚实纪律)。
type V13TaskRunErr struct {
	Task string
	Err  string
}

// V13RunsPage 是 Runs 页数据。
type V13RunsPage struct {
	Page
	// Unavailable:v1.3 数据源(事实面 / 制品仓库)未接入。
	Unavailable bool
	// LoadErr:Git 制品装载失败(如 HEAD 缺失、注册表缺失)。
	LoadErr string
	// ProjectID 来自 Git 注册表;空 = 未装载。
	ProjectID    string
	SourceCommit string

	// Runs 是全部任务的 Run 时间线(新到在前;跨任务合并的只读投影)。
	Runs []V13RunRow
	// TaskErrs 是按任务读取失败的显式清单(该任务的 Run 缺席有因可查)。
	TaskErrs []V13TaskRunErr

	// 计数从 Runs 行本身推导(汇总,不是第二状态源)。
	Total      int
	NRunning   int
	NCompleted int
	NFailed    int
	NEscalated int
	NCancelled int
}

// V13Runs 渲染运行(Runs)列表:实现 Run 过程事实的时间线投影(只读)。
func (s *Server) V13Runs(w http.ResponseWriter, r *http.Request) {
	page := V13RunsPage{Page: s.page(r, "运行", "v13-runs")}
	env, ok := s.v13env()
	if !ok {
		page.Unavailable = true
		v13Render(w, http.StatusOK, "v13_runs", page)
		return
	}
	s.v13RunsPage(r, env, &page)
	v13Render(w, http.StatusOK, "v13_runs", page)
}

func (s *Server) v13RunsPage(r *http.Request, env v13Env, page *V13RunsPage) {
	ctx := r.Context()
	g, err := openV13Git(env.repoPath)
	if err != nil {
		page.LoadErr = err.Error()
		return
	}
	snap, err := artifacts.Load(env.repoPath, g.headHash)
	if err != nil {
		page.LoadErr = err.Error()
		return
	}
	env.project = core.ProjectID(snap.ProjectID)
	page.ProjectID = snap.ProjectID
	page.SourceCommit = g.headHash

	// 清单经任务清单展开:逐任务走既有 CodegenRunsForTask 读路径。
	for _, ta := range snap.Tasks {
		runs, err := env.runs.CodegenRunsForTask(ctx, env.project, ta.Contract.ID)
		if err != nil {
			page.TaskErrs = append(page.TaskErrs, V13TaskRunErr{Task: ta.Contract.ID, Err: err.Error()})
			continue
		}
		for _, run := range runs {
			page.Runs = append(page.Runs, v13RunRow(ta.Contract.ID, ta.Contract.Title, run))
		}
	}

	// 新到在前:开始时刻降序,同刻按行 id 降序(确定性输出)。
	sort.SliceStable(page.Runs, func(i, j int) bool {
		if !page.Runs[i].startedAt.Equal(page.Runs[j].startedAt) {
			return page.Runs[i].startedAt.After(page.Runs[j].startedAt)
		}
		return page.Runs[i].id > page.Runs[j].id
	})

	// 汇总从行本身推导(展示统计,不是状态源)。
	for _, row := range page.Runs {
		page.Total++
		switch row.Status {
		case string(runstore.RunRunning):
			page.NRunning++
		case string(runstore.RunCompleted):
			page.NCompleted++
		case string(runstore.RunFailed):
			page.NFailed++
		case string(runstore.RunEscalated):
			page.NEscalated++
		case string(runstore.RunCancelled):
			page.NCancelled++
		}
	}
}

// v13RunRow 把一条 codegen_run 行投影成展示行(全部字段原样,不二次解释)。
func v13RunRow(task, title string, r runstore.Run) V13RunRow {
	observed, observedNote := v13ObservedView(r.RuntimeCapabilitySnapshot)
	row := V13RunRow{
		ID:            strconv.FormatInt(r.ID, 10),
		Task:          task,
		Title:         title,
		Status:        string(r.Status),
		StatusClass:   v13RunStatusClass(r.Status),
		Runtime:       r.Runtime,
		Model:         r.Model,
		Observed:      observed,
		ObservedNote:  observedNote,
		Iterations:    r.Iterations,
		Tokens:        v13RunTokens(r.InputTokens, r.OutputTokens),
		Started:       v13FormatTime(r.StartedAt),
		Finished:      v13FormatTime(r.FinishedAt),
		FailureReason: r.FailureReason,
		id:            r.ID,
		startedAt:     r.StartedAt,
	}
	if r.EscalatedFrom != 0 {
		row.EscalatedFrom = strconv.FormatInt(r.EscalatedFrom, 10)
	}
	if r.ResultPatchHash != "" {
		row.CandidateID = r.ResultPatchHash
	}
	return row
}

// v13RunStatusClass 是展示色映射(纯展示;状态文本保持行上原样):
// completed → ok;failed → bad;running/escalated → warn(进行中 / 被升级终止);
// 其余(cancelled 等)→ off。
func v13RunStatusClass(s runstore.RunStatus) string {
	switch s {
	case runstore.RunCompleted:
		return "st-ok"
	case runstore.RunFailed:
		return "st-bad"
	case runstore.RunRunning, runstore.RunEscalated:
		return "st-warn"
	default:
		return "st-off"
	}
}

// v13RunTokens 把用量计数格式化成「in N · out M」;两者都未上报(nil)返回空。
func v13RunTokens(in, out *int64) string {
	if in == nil && out == nil {
		return ""
	}
	inS, outS := "0", "0"
	if in != nil {
		inS = strconv.FormatInt(*in, 10)
	}
	if out != nil {
		outS = strconv.FormatInt(*out, 10)
	}
	return "in " + inS + " · out " + outS
}

// v13ObservedView 从能力快照 JSON 只读回放实际身份观察:键缺席 = 未观察
// (合法,零值降级);键在而解析失败 = 显式说明(快照矛盾不说成未观察)。
func v13ObservedView(snapshot json.RawMessage) (observed, note string) {
	if len(snapshot) == 0 {
		return "", ""
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(snapshot, &top); err != nil {
		return "", "实际身份快照无法解析(非 JSON 对象)"
	}
	raw, ok := top[v13ObservedIdentityKey]
	if !ok {
		return "", "" // 未观察:完全未观察 ≠ 期望值(Decision Lock 15)
	}
	var id v13ObservedIdentity
	if err := json.Unmarshal(raw, &id); err != nil {
		return "", "实际身份快照无法解析(observed_identity 形状不符)"
	}
	switch {
	case id.RuntimeName != "" && id.Model != "":
		observed = "实际 " + id.RuntimeName + " / " + id.Model
	case id.RuntimeName != "":
		observed = "实际 " + id.RuntimeName
	case id.Model != "":
		observed = "实际 " + id.Model
	}
	return observed, ""
}
