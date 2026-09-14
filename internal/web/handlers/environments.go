package handlers

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"sort"

	"tally/internal/environment"
	"tally/internal/jobs"
	"tally/internal/store"
)

// 环境页:状态从 environment_run 算出(只读,INV-3);
// 拉起 / 推倒 / 重建的 POST **只入队 job**,返回「已入队」—— 真正的执行与写事实面在 worker(INV-5)。

// EnvRow 是一个环境的状态行。
type EnvRow struct {
	Project    string
	Name       string
	Kind       string
	DefHash    string // 当前定义内容哈希(INV-4)短显
	Def        string // 定义仓库相对路径;空 → 定义缺失/解析失败,不提供动作
	StatusText string
	StateClass string
	LastAt     string
	Combo      string
	Note       string // 定义换了新版 / 定义已不在仓库 / 解析失败 等说明
}

// EnvironmentsPage 是环境状态页数据。
type EnvironmentsPage struct {
	Page
	Rows []EnvRow
}

// ListEnvironments 渲染环境状态页。
//
// 行 = 内容仓库里的定义(.tally/environments/*.yaml)与 environment_run 见过的环境的并集:
// 有定义的可行作;只剩历史记录(定义已移除)的只读展示,不给动作按钮。
func (s *Server) ListEnvironments(w http.ResponseWriter, r *http.Request) {
	data := EnvironmentsPage{Page: s.page(r, "环境", "environments"), Rows: s.envRows(r)}
	s.Render.Render(w, http.StatusOK, "environments", data)
}

// envAction 是三个环境动作 POST 的公共体:校验定义路径 → 入队 → PRG 回环境页。
func (s *Server) envAction(w http.ResponseWriter, r *http.Request, kind jobs.Kind) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单不合法: "+err.Error(), http.StatusBadRequest)
		return
	}
	def := r.FormValue("def")
	if !validDefPath(def) {
		redirectFlash(w, r, "/environments", "bad", "动作未入队:环境定义路径缺失或越界")
		return
	}
	s.enqueue(w, r, kind, jobs.EnvActionPayload{
		Project: r.FormValue("project"),
		Name:    r.FormValue("name"),
		Def:     def,
	}, "/environments")
}

// EnvUp 拉起入队。worker 执行 environment.Manager.Up 并写一行 environment_run。
func (s *Server) EnvUp(w http.ResponseWriter, r *http.Request) {
	s.envAction(w, r, jobs.KindEnvironmentUp)
}

// EnvDown 推倒入队(破坏性:模板里带警示色 + 二次确认)。
// worker 执行 Manager.Down 成功后写一行 status=torn_down。
func (s *Server) EnvDown(w http.ResponseWriter, r *http.Request) {
	s.envAction(w, r, jobs.KindEnvironmentDown)
}

// EnvRebuild 重建入队(破坏性:先推倒再拉起,模板里带警示色 + 二次确认)。
func (s *Server) EnvRebuild(w http.ResponseWriter, r *http.Request) {
	s.envAction(w, r, jobs.KindEnvironmentRebuild)
}

/* ---------- 状态行推导 ---------- */

// envRows 合并定义与最近拉起记录成状态行(确定性排序:项目、环境名)。
func (s *Server) envRows(r *http.Request) []EnvRow {
	latest := s.latestRuns(r)

	var rows []EnvRow
	seen := map[string]bool{}

	// 仓库里的定义 → 可行作的行
	if s.Content != nil && s.Content.RepoPath != "" {
		rels, err := s.Content.List(".tally/environments", ".yaml")
		if err != nil {
			slog.Error("列环境定义", "err", err)
		}
		for _, rel := range rels {
			def, perr := environment.ParseDefinition(filepath.Join(s.Content.RepoPath, filepath.FromSlash(rel)))
			if perr != nil {
				// 定义坏掉也要让人看见:给一行说明,不给动作(执行端同样解析不了)
				rows = append(rows, EnvRow{
					Project: "—", Name: filepath.Base(rel), Kind: "—",
					StatusText: "定义解析失败", StateClass: "st-bad", LastAt: "—",
					Note: perr.Error(),
				})
				continue
			}
			key := def.Project + "/" + def.Name
			seen[key] = true
			rows = append(rows, mergeEnvRow(def, rel, latest[key]))
		}
	}

	// 只剩历史记录、定义已不在仓库的环境 → 只读行
	keys := make([]string, 0, len(latest))
	for key := range latest {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		rows = append(rows, envRowFromRun(latest[key], "定义已不在内容仓库,无法动作"))
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Project != rows[j].Project {
			return rows[i].Project < rows[j].Project
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

// latestRuns 取每个环境最近一次拉起记录,key = project/name。读失败返回空(页面落空状态,不 500)。
func (s *Server) latestRuns(r *http.Request) map[string]store.EnvironmentRun {
	out := map[string]store.EnvironmentRun{}
	if s.Exec == nil {
		return out
	}
	runs, err := s.Exec.LatestEnvironmentRuns(r.Context())
	if err != nil {
		slog.Error("读最近拉起记录", "err", err)
		return out
	}
	for _, run := range runs {
		out[run.ProjectID+"/"+run.EnvName] = run
	}
	return out
}

// mergeEnvRow 把「定义 + 最近一次拉起记录」合成一个状态行。
func mergeEnvRow(def environment.Definition, rel string, run store.EnvironmentRun) EnvRow {
	row := EnvRow{
		Project: def.Project,
		Name:    def.Name,
		Kind:    string(def.Kind),
		DefHash: shortHash(def.Hash),
		Def:     rel,
	}
	if run.ID == 0 {
		row.StatusText, row.StateClass = "尚未拉起", "st-off"
		row.LastAt = "—"
		return row
	}
	row.StatusText, row.StateClass = envStatusView(run.Status)
	row.LastAt = fmtTime(run.StartedAt)
	row.Combo = comboBrief(run.VersionCombo)
	// 定义内容哈希对不上:上次拉起用的是旧定义(INV-4 的自然失效,不靠人标记)
	if run.DefinitionHash != def.Hash {
		row.Note = "上次拉起用的是旧版定义"
	}
	return row
}

// envRowFromRun 由一条历史记录造只读行(定义已不在仓库)。
func envRowFromRun(run store.EnvironmentRun, note string) EnvRow {
	text, class := envStatusView(run.Status)
	return EnvRow{
		Project: run.ProjectID, Name: run.EnvName, Kind: "—",
		DefHash:    shortHash(run.DefinitionHash),
		StatusText: text, StateClass: class,
		LastAt: fmtTime(run.StartedAt), Combo: comboBrief(run.VersionCombo),
		Note: note,
	}
}
