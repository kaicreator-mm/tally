package handlers

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"

	"tally/internal/deploy"
	"tally/internal/jobs"
)

// 部署页:历史从 deploy_run 读(只读,事实面只追加);
// 回滚 POST **只入队 job** —— worker 调工具(Kamal/Ansible/Compose)执行并写新 deploy_run(INV-5/6)。

// DeployRow 是一条部署记录。
type DeployRow struct {
	ID          int64
	Project     string
	Digest      string
	TargetEnv   string
	Tool        string
	StatusText  string
	StateClass  string
	Started     string
	ExitCode    string
	Combo       string
	RollbackDef string // 匹配的部署定义仓库相对路径;空 → 不渲染回滚按钮
}

// DeploysPage 是部署历史页数据。
type DeploysPage struct {
	Page
	Rows []DeployRow
}

// ListDeploys 渲染部署历史页(新→旧)。
func (s *Server) ListDeploys(w http.ResponseWriter, r *http.Request) {
	data := DeploysPage{Page: s.page(r, "部署", "deploys"), Rows: s.deployRows(r)}
	s.Render.Render(w, http.StatusOK, "deploys", data)
}

// DeployRollback 回滚入队(破坏性:模板里带警示色 + 二次确认)。
// payload 带目标 digest 与定义路径;worker 解析定义、覆盖 target_env/digest 后调工具回滚,
// 记一行 deploy_run(rolled_back / rollback_failed)。
func (s *Server) DeployRollback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单不合法: "+err.Error(), http.StatusBadRequest)
		return
	}
	def := r.FormValue("def")
	targetEnv := r.FormValue("target_env")
	digest := r.FormValue("digest")
	if !validDefPath(def) {
		redirectFlash(w, r, "/deploys", "bad", "回滚未入队:部署定义路径缺失或越界")
		return
	}
	if targetEnv == "" || digest == "" {
		redirectFlash(w, r, "/deploys", "bad", "回滚未入队:目标环境或目标制品缺失")
		return
	}
	s.enqueue(w, r, jobs.KindDeployRollback, jobs.DeployRollbackPayload{
		Project:   r.FormValue("project"),
		TargetEnv: targetEnv,
		Digest:    digest,
		Def:       def,
	}, "/deploys")
}

/* ---------- 历史行推导 ---------- */

// deployRows 把 deploy_run 记录转成展示行,并为「部署成功」的行匹配可用定义以放开回滚按钮。
func (s *Server) deployRows(r *http.Request) []DeployRow {
	if s.Exec == nil {
		return nil
	}
	runs, err := s.Exec.ListDeployRuns(r.Context())
	if err != nil {
		slog.Error("读部署历史", "err", err)
		return nil
	}
	defs := s.deployDefs()
	rows := make([]DeployRow, 0, len(runs))
	for _, run := range runs {
		text, class := deployStatusView(run.Status)
		exit := "—"
		if run.ExitCode.Valid {
			exit = strconv.Itoa(int(run.ExitCode.Int32))
		}
		row := DeployRow{
			ID: run.ID, Project: run.ProjectID, Digest: run.ArtifactDigest,
			TargetEnv: run.TargetEnv, Tool: run.Tool,
			StatusText: text, StateClass: class,
			Started: fmtTime(run.StartedAt), ExitCode: exit,
			Combo: comboBrief(run.VersionCombo),
		}
		// 只有「部署成功」的版本值得回滚回去;且得有该环境的部署定义(执行回滚要靠它调工具)。
		if run.Status == deploy.StatusDeployed {
			row.RollbackDef = defs[run.ProjectID+"/"+run.TargetEnv]
		}
		rows = append(rows, row)
	}
	return rows
}

// deployDefs 扫内容仓库 .tally/deploys/*.yaml,建 project/target_env → 相对路径 的索引。
// 解析坏掉的定义跳过并记日志(界面上表现为该环境没有可用回滚定义)。
func (s *Server) deployDefs() map[string]string {
	out := map[string]string{}
	if s.Content == nil || s.Content.RepoPath == "" {
		return out
	}
	rels, err := s.Content.List(".tally/deploys", ".yaml")
	if err != nil {
		slog.Error("列部署定义", "err", err)
		return out
	}
	for _, rel := range rels {
		def, perr := deploy.ParseDefinition(filepath.Join(s.Content.RepoPath, filepath.FromSlash(rel)))
		if perr != nil {
			slog.Warn("部署定义解析失败,回滚不可用", "def", rel, "err", perr)
			continue
		}
		out[def.Project+"/"+def.TargetEnv] = rel
	}
	return out
}
