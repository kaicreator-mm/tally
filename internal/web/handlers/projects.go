package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// 项目登记是 git 里的 projects/<slug>.md,格式:
//
//	# <名称>
//
//	<一句话说明>

// ProjectView 是一个项目的展示数据。
type ProjectView struct {
	Slug        string
	Name        string
	Description string
}

func projectPath(slug string) string { return "projects/" + slug + ".md" }

// parseProject 从 markdown 提取名称(H1)与说明(第一段非标题正文)。
func parseProject(slug, content string) ProjectView {
	p := ProjectView{Slug: slug}
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if p.Name == "" && strings.HasPrefix(t, "# ") {
			p.Name = strings.TrimSpace(strings.TrimPrefix(t, "# "))
			continue
		}
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if p.Description == "" {
			p.Description = t
		}
	}
	if p.Name == "" {
		p.Name = slug
	}
	return p
}

func serializeProject(name, desc string) string {
	return fmt.Sprintf("# %s\n\n%s\n", name, strings.TrimSpace(desc))
}

// loadProject 读并解析一个项目;不存在返回 ErrNotFound。
func (s *Server) loadProject(slug string) (ProjectView, error) {
	content, err := s.Content.Read(projectPath(slug))
	if err != nil {
		return ProjectView{}, err
	}
	return parseProject(slug, content), nil
}

// Index 是总览:四个内容入口的计数 + 任务算出状态入口。
func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Page
		ProjectCount     int
		RequirementCount int
		DocCount         int
		WikiCount        int
		TaskCount        int
	}{Page: s.page(r, "总览", "index")}
	data.ProjectCount, _ = s.count("projects")
	data.RequirementCount, _ = s.count(".tally/requirements")
	data.DocCount, _ = s.count("docs")
	data.WikiCount, _ = s.count("wiki")
	data.TaskCount = len(s.taskRows(r))
	s.Render.Render(w, http.StatusOK, "index", data)
}

// count 统计某目录下 .md 文件数(出错按 0)。
func (s *Server) count(dir string) (int, error) {
	files, err := s.Content.List(dir, ".md")
	if err != nil {
		return 0, err
	}
	return len(files), nil
}

// ProjectsPage 是项目列表页数据。
type ProjectsPage struct {
	Page
	Projects []ProjectView
}

// ListProjects 列出全部项目。
func (s *Server) ListProjects(w http.ResponseWriter, r *http.Request) {
	files, err := s.Content.List("projects", ".md")
	if err != nil {
		s.serverError(w, r, "读项目列表", err)
		return
	}
	data := ProjectsPage{Page: s.page(r, "项目", "projects")}
	for _, f := range files {
		slug := strings.TrimSuffix(strings.TrimPrefix(f, "projects/"), ".md")
		if p, err := s.loadProject(slug); err == nil {
			data.Projects = append(data.Projects, p)
		}
	}
	s.Render.Render(w, http.StatusOK, "projects", data)
}

// ProjectFormPage 是新建/编辑项目共用表单数据。
type ProjectFormPage struct {
	Page
	Slug        string
	Name        string
	Description string
	IsNew       bool
	Action      string
	FormError   string
}

// NewProjectForm 渲染新建项目表单。
func (s *Server) NewProjectForm(w http.ResponseWriter, r *http.Request) {
	data := ProjectFormPage{
		Page:   s.page(r, "新建项目", "projects"),
		IsNew:  true,
		Action: "/projects",
	}
	s.Render.Render(w, http.StatusOK, "project_form", data)
}

// CreateProject 建项目:写 git projects/<slug>.md 并 commit。
func (s *Server) CreateProject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	slug := strings.TrimSpace(r.FormValue("slug"))
	name := strings.TrimSpace(r.FormValue("name"))
	desc := strings.TrimSpace(r.FormValue("description"))

	if !validSlug(slug) {
		s.renderProjectFormError(w, r, true, slug, name, desc, "标识只能用小写字母、数字、连字符(如 forge)")
		return
	}
	if name == "" {
		s.renderProjectFormError(w, r, true, slug, name, desc, "名称不能为空")
		return
	}
	if _, err := s.Content.Read(projectPath(slug)); err == nil {
		s.renderProjectFormError(w, r, true, slug, name, desc, "标识 "+slug+" 已存在,换一个")
		return
	} else if !errors.Is(err, ErrNotFound) {
		s.serverError(w, r, "检查项目", err)
		return
	}

	if _, err := s.Content.Commit(projectPath(slug), []byte(serializeProject(name, desc)), "feat: 新建项目 "+name); err != nil {
		s.serverError(w, r, "写入项目", err)
		return
	}
	redirectFlash(w, r, "/projects", "ok", "项目 "+name+" 已建,已提交到 git")
}

// renderProjectFormError 以 400 重绘项目表单并显示错误(保留用户已填内容)。
func (s *Server) renderProjectFormError(w http.ResponseWriter, r *http.Request, isNew bool, slug, name, desc, msg string) {
	action := "/projects"
	if !isNew {
		action = "/projects/" + slug
	}
	data := ProjectFormPage{
		Page: s.page(r, "新建项目", "projects"),
		Slug: slug, Name: name, Description: desc,
		IsNew: isNew, Action: action, FormError: msg,
	}
	s.Render.Render(w, http.StatusBadRequest, "project_form", data)
}

// ProjectDetailPage 是项目详情页数据。
type ProjectDetailPage struct {
	Page
	Project          ProjectView
	RequirementCount int
	LastCommit       string
}

// ProjectDetail 展示一个项目及其名下内容入口。
func (s *Server) ProjectDetail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validSlug(slug) {
		http.NotFound(w, r)
		return
	}
	p, err := s.loadProject(slug)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, r, "读项目", err)
		return
	}
	reqs, _ := s.Content.List(".tally/requirements/"+slug, ".md")
	data := ProjectDetailPage{
		Page:             s.page(r, p.Name, "projects"),
		Project:          p,
		RequirementCount: len(reqs),
		LastCommit:       s.Content.LastCommit(projectPath(slug)),
	}
	s.Render.Render(w, http.StatusOK, "project_detail", data)
}

// EditProjectForm 渲染编辑项目表单。
func (s *Server) EditProjectForm(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validSlug(slug) {
		http.NotFound(w, r)
		return
	}
	p, err := s.loadProject(slug)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, r, "读项目", err)
		return
	}
	data := ProjectFormPage{
		Page: s.page(r, "编辑项目", "projects"),
		Slug: p.Slug, Name: p.Name, Description: p.Description,
		IsNew:  false,
		Action: "/projects/" + slug,
	}
	s.Render.Render(w, http.StatusOK, "project_form", data)
}

// UpdateProject 改项目:写 git 并 commit(slug 不可改)。
func (s *Server) UpdateProject(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validSlug(slug) {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	desc := strings.TrimSpace(r.FormValue("description"))
	if name == "" {
		s.renderProjectFormError(w, r, false, slug, name, desc, "名称不能为空")
		return
	}
	if _, err := s.Content.Read(projectPath(slug)); errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if _, err := s.Content.Commit(projectPath(slug), []byte(serializeProject(name, desc)), "feat: 更新项目 "+name); err != nil {
		s.serverError(w, r, "写入项目", err)
		return
	}
	redirectFlash(w, r, "/projects/"+slug, "ok", "已保存并提交到 git")
}
