package handlers

import (
	"context"
	"log/slog"
	"net/http"
)

// 本页是遗留的「项目执行摘要」:一个项目一行,栏目全部从执行记录
// (build_run / deploy_run / failure_event)与需求层条款覆盖推导,只读、无控件。
//
// P0-22 语义边界:它不是 v1.3 的工程 Task(TaskContract,PRD §7 / §36)域,
// 不呈现也不推导 Task 完成 / VERIFIED / APPROVED 语义;TaskContract 域在 P1 落地。
// 在那之前本页只作为执行/项目摘要保留。因此用户可见措辞一律是记录式表述
// (「有记录 / 有覆盖」),不再出现「已合并 / 测试通过」这类可被误读为完成判定的词。
//
// 结构里的字段名(Merged / Tested 等)是历史遗留命名,外部测试与调用方仍在引用,
// 为不越界改动而保留;各字段的真实含义见字段注释与 taskstatus.go 的推导规则。
// 摘要完全从事实面推导、可重建,这里不读任何手动状态字段 —— 没有这种字段可读。

// TaskRow 是一个项目的执行摘要行(五栏,纯展示;不含任何 Task 完成语义)。
type TaskRow struct {
	Project  string
	Title    string
	Merged   string // 构建记录:有记录 ⟹ 存在该项目的 build_run(不判定 Git 合并,不判定 Task 完成)
	Built    string // 末次构建:最近一次 build_run 的通过 / 失败
	Tested   string // 条款覆盖:需求层条款被测试引用 ⟹ 有覆盖(不是验证通过 / VERIFIED)
	Deployed string // 末次部署:最近一次 deploy_run 的成功 / 失败
	Stable   string // 部署后:最近一次部署之后有无新的失败记录(failure_event)
}

// TaskSource 提供项目执行摘要行。实现方从执行链推导,不读任何手动状态字段。
type TaskSource interface {
	TaskRows(ctx context.Context) ([]TaskRow, error)
}

// TasksPage 是执行摘要页数据。模板里不渲染任何 checkbox / 状态表单。
type TasksPage struct {
	Page
	Rows []TaskRow
}

// taskRows 取算出的摘要行;未接入来源或出错时返回空(不 500:没执行数据是正常状态)。
func (s *Server) taskRows(r *http.Request) []TaskRow {
	if s.Tasks == nil {
		return nil
	}
	rows, err := s.Tasks.TaskRows(r.Context())
	if err != nil {
		slog.Error("算执行摘要行", "err", err)
		return nil
	}
	return rows
}

// ListTasks 渲染项目执行摘要页(只读,无控件;不是 TaskContract 完成视图)。
func (s *Server) ListTasks(w http.ResponseWriter, r *http.Request) {
	data := TasksPage{Page: s.page(r, "执行摘要", "tasks"), Rows: s.taskRows(r)}
	s.Render.Render(w, http.StatusOK, "tasks", data)
}
