package handlers

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/deploy"
	"tally/internal/project"
	"tally/internal/store"
)

// 项目执行摘要(P0-22 重述语义):一个项目一行,栏目全部从执行记录推导 ——
//
//	构建记录  build_run 有该项目的记录(只说明存在构建记录;不判定 Git 合并,不判定 Task 完成)
//	末次构建  最近一次 build_run 的退出码 / 状态
//	条款覆盖  需求层覆盖率:test_clause(@covers)引用了本项目当前条款 ⟹ 有覆盖(不是验证通过 / VERIFIED)
//	末次部署  最近一次 deploy_run 的结果
//	部署后    最近一次部署之后是否又有失败(failure_event)
//
// 本页是执行/项目摘要,不是 v1.3 TaskContract(PRD §7)完成视图;界面上没有任何能改它的
// 控件,这里也不读任何手动状态字段 —— 没有这种字段可读。Task 域在 P1 落地。

// ExecTaskSource 从执行链算出项目执行摘要行(TaskSource 的实现)。db 为 *store.Queries。
type ExecTaskSource struct {
	db *store.Queries
}

// NewExecTaskSource 构造从执行链算摘要行的来源。
func NewExecTaskSource(db *store.Queries) *ExecTaskSource {
	return &ExecTaskSource{db: db}
}

// TaskRows 算出全部项目的摘要行。任一查询出错返回错误(调用方落空列表,不 500)。
func (s *ExecTaskSource) TaskRows(ctx context.Context) ([]TaskRow, error) {
	builds, err := s.db.TaskBuildSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("算构建记录栏: %w", err)
	}
	stabs, err := s.db.TaskStability(ctx)
	if err != nil {
		return nil, fmt.Errorf("算部署栏: %w", err)
	}
	clauseProjects, err := s.db.ListClauseProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("列需求项目: %w", err)
	}
	// 条款覆盖栏:有需求层且当前条款被测试引用 ⟹ 有覆盖。逐项目算覆盖率(规模为自家十来个项目,够用)。
	// 注意这是覆盖事实,不是验证结论 —— 不呈现「测试通过」。
	clauseCovered := map[string]bool{}
	for _, p := range clauseProjects {
		cov, cerr := project.ClauseCoverage(ctx, s.db, core.ProjectID(p))
		if cerr != nil {
			return nil, fmt.Errorf("算条款覆盖率(%s): %w", p, cerr)
		}
		if cov.Total > 0 && cov.Covered > 0 {
			clauseCovered[p] = true
		}
	}
	return computeTaskRows(builds, stabs, clauseProjects, clauseCovered), nil
}

// taskAccum 是一个项目在推导过程中的累积;空串字段最后落「无记录」。
type taskAccum struct {
	project  string
	merged   string
	built    string
	tested   string
	deployed string
	stable   string
}

// computeTaskRows 把构建概要、部署稳定性、需求项目与「条款有覆盖」集合纯合成摘要行。
// 纯函数,不碰数据库,单测钉住各栏的推导规则与措辞边界(不出现完成判定式措辞)。
func computeTaskRows(builds []store.TaskBuildSummaryRow, stabs []store.TaskStabilityRow,
	clauseProjects []string, clauseCovered map[string]bool) []TaskRow {
	acc := map[string]*taskAccum{}
	get := func(p string) *taskAccum {
		if a, ok := acc[p]; ok {
			return a
		}
		a := &taskAccum{project: p}
		acc[p] = a
		return a
	}
	// 只有需求层的项目也进表(即便还没构建/部署):那是「已立项、未动工」的项目。
	for _, p := range clauseProjects {
		get(p)
	}
	for _, b := range builds {
		a := get(b.ProjectID)
		a.merged = "有记录" // 存在 build_run 记录;不再据此宣称「已合并」
		a.built = buildOutcome(b.Status, b.ExitCode)
	}
	for _, d := range stabs {
		a := get(d.ProjectID)
		a.deployed = deployOutcome(d.LastDeployStatus)
		a.stable = "稳定"
		if d.HasPostDeployFailure {
			a.stable = "有复发"
		}
	}
	for p := range clauseCovered {
		get(p).tested = "有覆盖" // 覆盖事实,不是「测试通过」
	}

	rows := make([]TaskRow, 0, len(acc))
	for _, a := range acc {
		rows = append(rows, TaskRow{
			Project:  a.project,
			Title:    a.project,
			Merged:   orNone(a.merged),
			Built:    orNone(a.built),
			Tested:   orNone(a.tested),
			Deployed: orNone(a.deployed),
			Stable:   orNone(a.stable),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Project < rows[j].Project })
	return rows
}

// orNone 把推导不出的空栏落「无记录」(不是空串,界面要显示得出)。
func orNone(s string) string {
	if s == "" {
		return "无记录"
	}
	return s
}

// buildOutcome 从构建的退出码 / 状态推「通过 / 失败」。这是末次 build_run 的结果记录,
// 不是 Task 验证结论。退出码是更可靠的机器信号:有效且为 0 → 通过,有效非 0 → 失败;
// 退出码缺失时退回状态串判断,都不认得落「无记录」。
func buildOutcome(status string, exit pgtype.Int4) string {
	if exit.Valid {
		if exit.Int32 == 0 {
			return "通过"
		}
		return "失败"
	}
	switch status {
	case "passed", "success", "ok", "green":
		return "通过"
	case "failed", "failure", "error", "red":
		return "失败"
	default:
		return "无记录"
	}
}

// deployOutcome 从部署状态推「成功 / 失败」(措辞与部署页一致)。
func deployOutcome(status string) string {
	switch status {
	case deploy.StatusDeployed, deploy.StatusRolledBack:
		return "成功"
	case deploy.StatusFailed, deploy.StatusRollbackFailed:
		return "失败"
	default:
		return "无记录"
	}
}
