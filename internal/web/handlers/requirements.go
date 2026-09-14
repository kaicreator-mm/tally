package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"tally/internal/requirement"
)

// 需求条款是 git 里的 .tally/requirements/<project>/<file>.md(EARS markdown)。
// 编辑落 git;解析 / lint / 内容哈希复用 requirement 包,仅用于只读展示。

func requirementPath(project, file string) string {
	return ".tally/requirements/" + project + "/" + file
}

// ReqFileView 是需求文件列表的一项。
type ReqFileView struct {
	Project string
	File    string
	Title   string
	Clauses int
}

// RequirementsPage 是需求列表页数据。
type RequirementsPage struct {
	Page
	Project string // 当前过滤的项目("" = 全部)
	Files   []ReqFileView
}

// ListRequirements 列出需求文件,可按 ?project= 过滤。
func (s *Server) ListRequirements(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("project")
	files, err := s.Content.List(".tally/requirements", ".md")
	if err != nil {
		s.serverError(w, r, "读需求列表", err)
		return
	}
	data := RequirementsPage{Page: s.page(r, "需求", "requirements"), Project: filter}
	for _, f := range files {
		// f 形如 .tally/requirements/<project>/<file>.md
		rest := strings.TrimPrefix(f, ".tally/requirements/")
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) != 2 {
			continue
		}
		project, file := parts[0], parts[1]
		if filter != "" && project != filter {
			continue
		}
		v := ReqFileView{Project: project, File: file}
		if content, err := s.Content.Read(f); err == nil {
			doc := requirement.Parse(content)
			v.Title = doc.Title
			v.Clauses = len(doc.Clauses)
		}
		data.Files = append(data.Files, v)
	}
	s.Render.Render(w, http.StatusOK, "requirements", data)
}

// CreateRequirement 新建需求文件:写 git .tally/requirements/<project>/<file> 并 commit。
func (s *Server) CreateRequirement(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	project := strings.TrimSpace(r.FormValue("project"))
	file := strings.TrimSpace(r.FormValue("file"))
	content := r.FormValue("content")

	if !validSlug(project) {
		redirectFlash(w, r, "/requirements", "bad", "项目标识只能用小写字母、数字、连字符")
		return
	}
	if !validFileName(file) {
		redirectFlash(w, r, "/requirements?project="+project, "bad", "文件名须以 .md 结尾,只含字母数字与 - _ .")
		return
	}
	if strings.TrimSpace(content) == "" {
		redirectFlash(w, r, "/requirements?project="+project, "bad", "内容不能为空")
		return
	}
	if _, err := s.Content.Read(requirementPath(project, file)); err == nil {
		redirectFlash(w, r, "/requirements?project="+project, "bad", file+" 已存在")
		return
	} else if !errors.Is(err, ErrNotFound) {
		s.serverError(w, r, "检查需求文件", err)
		return
	}
	if _, err := s.Content.Commit(requirementPath(project, file), []byte(content), "feat: 新建需求 "+project+"/"+file); err != nil {
		s.serverError(w, r, "写入需求", err)
		return
	}
	redirectFlash(w, r, "/requirements/"+project+"/"+file, "ok", "已写入 git")
}

// ClauseView / GateView / IssueView 是需求编辑页的只读展示数据。
type ClauseView struct {
	ID        string
	Title     string
	Statement string
	Mechanism string
	Criteria  []string
	Hash      string
}

type GateView struct {
	Level string
	OK    bool
	Count int
}

type IssueView struct {
	Level   string
	Clause  string
	Message string
}

// RequirementEditPage 是需求编辑页数据。
type RequirementEditPage struct {
	Page
	Project   string
	File      string
	Content   string
	Clauses   []ClauseView
	Gates     []GateView
	Issues    []IssueView
	FormError string
}

// buildRequirementEdit 从正文构建编辑页的只读展示(解析 + 五闸 + 内容哈希)。
func buildRequirementEdit(page Page, project, file, content string, formErr string) RequirementEditPage {
	doc := requirement.Parse(content)
	data := RequirementEditPage{
		Page: page, Project: project, File: file, Content: content, FormError: formErr,
	}
	for _, c := range doc.Clauses {
		data.Clauses = append(data.Clauses, ClauseView{
			ID: c.ID, Title: c.Title, Statement: c.Statement, Mechanism: c.Mechanism,
			Criteria: c.Criteria, Hash: shortHash(string(requirement.ContentHash(c))),
		})
	}
	issues := requirement.Lint(doc)
	counts := map[string]int{}
	for _, iss := range issues {
		counts[iss.Level]++
		data.Issues = append(data.Issues, IssueView{Level: iss.Level, Clause: iss.Clause, Message: iss.Message})
	}
	for _, lvl := range []string{"L1", "L2", "L3", "L4", "L5"} {
		data.Gates = append(data.Gates, GateView{Level: lvl, OK: counts[lvl] == 0, Count: counts[lvl]})
	}
	return data
}

// shortHash 把 sha256:<hex> 截成前 6 位十六进制,便于展示。
func shortHash(h string) string {
	h = strings.TrimPrefix(h, "sha256:")
	if len(h) > 6 {
		h = h[:6]
	}
	return h
}

// RequirementForm 渲染需求编辑页(解析结构 + 五闸 + 正文编辑区)。
func (s *Server) RequirementForm(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	file := chi.URLParam(r, "file")
	if !validSlug(project) || !validFileName(file) {
		http.NotFound(w, r)
		return
	}
	content, err := s.Content.Read(requirementPath(project, file))
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, r, "读需求", err)
		return
	}
	page := s.page(r, fmt.Sprintf("%s / %s", project, file), "requirements")
	data := buildRequirementEdit(page, project, file, content, "")
	s.Render.Render(w, http.StatusOK, "requirement_edit", data)
}

// UpdateRequirement 保存需求正文:写 git 并 commit。
func (s *Server) UpdateRequirement(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	file := chi.URLParam(r, "file")
	if !validSlug(project) || !validFileName(file) {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	content := r.FormValue("content")
	if _, err := s.Content.Read(requirementPath(project, file)); errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if _, err := s.Content.Commit(requirementPath(project, file), []byte(content), "feat: 更新需求 "+project+"/"+file); err != nil {
		s.serverError(w, r, "写入需求", err)
		return
	}
	redirectFlash(w, r, "/requirements/"+project+"/"+file, "ok", "已保存并提交到 git")
}
