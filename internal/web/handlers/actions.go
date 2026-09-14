package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/deploy"
	"tally/internal/environment"
	"tally/internal/jobs"
	"tally/internal/store"
)

/* ---------- 动作入队(界面写边界:触发动作 = 入队 jobs) ---------- */

// JobQueue 是 handlers 对动作队列的要求(在使用方定义接口)。*jobs.Queue 结构上满足
// (Enqueue 签名一致)。经接口注入:单测用替身断言「入了什么队」,nil 表示队列未接入。
type JobQueue interface {
	Enqueue(ctx context.Context, kind jobs.Kind, payload any) (int64, error)
}

// ExecSource 是执行引擎四个页面(环境/部署/失败/指纹)的读取来源(在使用方定义接口)。
// *store.Queries 结构上满足。nil → 各页渲染空状态行动提示,不 500。
//
// 全部是对事实面/索引面的只读查询:handler 侧没有任何写入 ——
// 事实面(environment_run/deploy_run)只由 worker 执行动作时 INSERT(INV-5)。
type ExecSource interface {
	LatestEnvironmentRuns(ctx context.Context) ([]store.EnvironmentRun, error)
	ListDeployRuns(ctx context.Context) ([]store.DeployRun, error)
	GetFingerprintStat(ctx context.Context, fpExact string) (store.FingerprintStat, error)
	ListFingerprintStats(ctx context.Context) ([]store.FingerprintStat, error)
	ListFingerprintStatsByFuzzy(ctx context.Context, fpFuzzy string) ([]store.FingerprintStat, error)
	ListLatestFailureExcerpts(ctx context.Context) ([]store.ListLatestFailureExcerptsRow, error)
	CountFailuresByDay(ctx context.Context) ([]store.CountFailuresByDayRow, error)
	ListDependencyChangesRecent(ctx context.Context) ([]store.DependencyChange, error)
	ViewFingerprintImpact(ctx context.Context, fpExact string) ([]store.ViewFingerprintImpactRow, error)
	ListFailureEventsByFingerprint(ctx context.Context, fpExact string) ([]store.ListFailureEventsByFingerprintRow, error)
}

// enqueue 是动作 POST 的唯一出口:只往 job 表 INSERT 一条,然后带「已入队」导航回去(PRG)。
//
// 界面绝不写事实面(INV-5):environment_run/deploy_run 由 worker 执行这些 job 时写。
// 队列未接入 / 入队失败 → 给明确原因与下一步(06-ui-spec:失败要说明白),不假装成功。
func (s *Server) enqueue(w http.ResponseWriter, r *http.Request, kind jobs.Kind, payload any, backPath string) {
	if s.Queue == nil {
		redirectFlash(w, r, backPath, "bad", "动作未入队:队列未接入,检查 server 是否连上了 jobs 队列")
		return
	}
	id, err := s.Queue.Enqueue(r.Context(), kind, payload)
	if err != nil {
		slog.Error("动作入队", "kind", string(kind), "err", err)
		redirectFlash(w, r, backPath, "bad", "动作未入队: "+err.Error())
		return
	}
	redirectFlash(w, r, backPath, "ok",
		fmt.Sprintf("已入队(job#%d)。worker 执行时会写一条新记录,稍后刷新即可看到结果", id))
}

// validDefPath 校验表单来的定义路径是安全的仓库相对路径(防穿越、防绝对路径)。
// 定义路径来自界面,是不可信输入:执行端会按内容仓库根目录拼它。
func validDefPath(s string) bool {
	if s == "" || strings.HasPrefix(s, "/") || strings.Contains(s, `\`) {
		return false
	}
	for _, seg := range strings.Split(s, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return strings.HasSuffix(s, ".yaml") || strings.HasSuffix(s, ".yml")
}

/* ---------- 展示辅助(纯函数,单测钉住) ---------- */

// envStatusView 从 environment_run.status 推出展示文案与状态色。状态是算出来的(INV-3)。
// torn_down 由 Manager.Down 成功后写入;UI 渲染为「已停止」。
func envStatusView(status string) (text, class string) {
	switch status {
	case environment.StatusUp:
		return "在跑", "st-ok"
	case environment.StatusFailed:
		return "上次拉起失败", "st-bad"
	case environment.StatusTornDown:
		return "已停止", "st-off"
	default:
		return "尚未拉起", "st-off"
	}
}

// deployStatusView 从 deploy_run.status 推出展示文案与状态色。
func deployStatusView(status string) (text, class string) {
	switch status {
	case deploy.StatusDeployed:
		return "部署成功", "st-ok"
	case deploy.StatusFailed:
		return "部署失败", "st-bad"
	case deploy.StatusRolledBack:
		return "已回滚", "st-ok"
	case deploy.StatusRollbackFailed:
		return "回滚失败", "st-bad"
	default:
		return status, "st-off"
	}
}

// classificationView 从 fingerprint_stat.status 推出处理标签(措辞与 failure 五种出口一致)。
// fixed 且已漂移(修复验证后又复发)→「修复可能过期」,赭石色:不能直接套用,只作线索。
func classificationView(status string, drifted bool) (text, class string) {
	if status == "fixed" && drifted {
		return "修复可能过期", "warn"
	}
	switch status {
	case "fixed":
		return "已验证修复", "ok"
	case "flaky":
		return "偶发,不入库", ""
	case "upstream":
		return "等上游", "warn"
	case "infra":
		return "基础设施", ""
	case "spec_defect":
		return "需求缺陷", "bad"
	default:
		return "未诊断", ""
	}
}

// sourceText 失败来源的人读文案。
func sourceText(source string) string {
	switch source {
	case "build":
		return "构建"
	case "deploy":
		return "部署"
	case "environment":
		return "环境"
	default:
		return source
	}
}

// rankValue 是修复排行分:出现次数 × 影响项目数 —— 该先修哪个。
func rankValue(count int64, projects int) int64 {
	n := int64(projects)
	if n < 1 {
		n = 1
	}
	return count * n
}

// comboBrief 把 version_combo JSON 压成一行概要,供表格窄列展示。
// 解码失败(空 / 非 JSON)落空串;排序与截断规则见 formatComboMap。
func comboBrief(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return ""
	}
	return formatComboMap(m)
}

// formatComboMap 把版本组合压成一行:键排序后 `k=v` 空格连接,超 80 截断加省略号。
// 排序 + 截断规则的单一来源 —— comboBrief(JSON 输入)与 comboText(core.VersionCombo 输入)共用,
// 改截断长度只动这里。空组合落空串。
func formatComboMap(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+m[k])
	}
	s := strings.Join(parts, " ")
	const max = 80
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}

// fmtTime 页面统一时刻格式(月-日 时:分);无效时刻落「—」。
func fmtTime(t pgtype.Timestamptz) string {
	if !t.Valid {
		return "—"
	}
	return t.Time.Local().Format("01-02 15:04")
}

// verifiedAtText 格式化验证时刻(年-月-日 时:分);零时刻落「—」。
func verifiedAtText(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("2006-01-02 15:04")
}

// excerptFirstLine 取脱敏片段的第一行并截断,供列表摘要列。
func excerptFirstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	const max = 80
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}

/* ---------- 失败时间线 × 依赖升级对齐 ---------- */

// timelineWindowDays 是失败时间线的窗口(天):与两个只读查询的窗口一致。
const timelineWindowDays = 90

// TimelineDay 是时间线上的一天:当日失败数 + 是否有依赖变更(及内容)。
type TimelineDay struct {
	Label    string // 07-21
	Full     string // 2026-07-21(title 与轴标签)
	Failures int64
	Pct      int // 柱高百分比;0 次 → 零刻度(2px 灰)
	Deps     []string
	DepTitle string
	HasDep   bool
}

// buildTimeline 把按天失败数与依赖变更对齐到最近 days 天的同一日刻度上(纯函数,可用固定时刻测)。
//
// 这条对齐关系是这个视图存在的唯一理由:哪根失败柱子旁边有依赖刻度,哪次升级就可能是元凶。
func buildTimeline(failures []store.CountFailuresByDayRow, deps []store.DependencyChange, now time.Time, days int) []TimelineDay {
	failByDay := make(map[string]int64, len(failures))
	for _, f := range failures {
		if f.Day.Valid {
			failByDay[f.Day.Time.UTC().Format("2006-01-02")] = f.Failures
		}
	}
	depByDay := map[string][]string{}
	for _, d := range deps {
		if !d.ChangedAt.Valid {
			continue
		}
		key := d.ChangedAt.Time.UTC().Format("2006-01-02")
		desc := d.Package
		if d.ToVersion.Valid {
			desc += " → " + d.ToVersion.String
		}
		depByDay[key] = append(depByDay[key], d.ProjectID+": "+desc)
	}

	var max int64
	for _, n := range failByDay {
		if n > max {
			max = n
		}
	}

	out := make([]TimelineDay, days)
	start := now.UTC().AddDate(0, 0, -(days - 1))
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		n := failByDay[key]
		pct := 0
		if n > 0 {
			pct = int(n * 100 / max)
			if pct < 4 {
				pct = 4
			}
		}
		dayDeps := depByDay[key]
		out[i] = TimelineDay{
			Label:    day.Format("01-02"),
			Full:     key,
			Failures: n,
			Pct:      pct,
			Deps:     dayDeps,
			DepTitle: strings.Join(dayDeps, "; "),
			HasDep:   len(dayDeps) > 0,
		}
	}
	return out
}
