package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// 文档是 git 里的 docs/<slug>.md。渲染是轻量只读视图(见 web.renderMarkdown),编辑落回 git。

func docPath(slug string) string { return "docs/" + slug + ".md" }

// docTitle 取 markdown 的 H1 作标题;没有就用 slug。
func docTitle(slug, content string) string {
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "# "))
		}
	}
	return slug
}

// DocView 是文档列表的一项。
type DocView struct {
	Slug  string
	Title string
}

// DocsPage 是文档列表页数据。
type DocsPage struct {
	Page
	Docs []DocView
}

// ListDocs 列出全部文档。
func (s *Server) ListDocs(w http.ResponseWriter, r *http.Request) {
	files, err := s.Content.List("docs", ".md")
	if err != nil {
		s.serverError(w, r, "读文档列表", err)
		return
	}
	data := DocsPage{Page: s.page(r, "文档", "docs")}
	for _, f := range files {
		slug := strings.TrimSuffix(strings.TrimPrefix(f, "docs/"), ".md")
		title := slug
		if content, err := s.Content.Read(f); err == nil {
			title = docTitle(slug, content)
		}
		data.Docs = append(data.Docs, DocView{Slug: slug, Title: title})
	}
	s.Render.Render(w, http.StatusOK, "docs", data)
}

// DocEditPage 是文档编辑页数据(新建/编辑共用)。
type DocEditPage struct {
	Page
	Slug      string
	Content   string
	IsNew     bool
	Action    string
	FormError string
}

// NewDocForm 渲染新建文档表单。
func (s *Server) NewDocForm(w http.ResponseWriter, r *http.Request) {
	data := DocEditPage{Page: s.page(r, "新建文档", "docs"), IsNew: true, Action: "/docs"}
	s.Render.Render(w, http.StatusOK, "doc_edit", data)
}

// CreateDoc 新建文档:写 git docs/<slug>.md 并 commit。
func (s *Server) CreateDoc(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	slug := strings.TrimSpace(r.FormValue("slug"))
	content := r.FormValue("content")
	if !validSlug(slug) {
		s.renderDocFormError(w, r, true, slug, content, "标识只能用小写字母、数字、连字符")
		return
	}
	if _, err := s.Content.Read(docPath(slug)); err == nil {
		s.renderDocFormError(w, r, true, slug, content, "标识 "+slug+" 已存在")
		return
	} else if !errors.Is(err, ErrNotFound) {
		s.serverError(w, r, "检查文档", err)
		return
	}
	if _, err := s.Content.Commit(docPath(slug), []byte(content), "docs: 新建文档 "+slug); err != nil {
		s.serverError(w, r, "写入文档", err)
		return
	}
	redirectFlash(w, r, "/docs/"+slug, "ok", "已写入 git")
}

func (s *Server) renderDocFormError(w http.ResponseWriter, r *http.Request, isNew bool, slug, content, msg string) {
	action := "/docs"
	if !isNew {
		action = "/docs/" + slug
	}
	data := DocEditPage{
		Page: s.page(r, "新建文档", "docs"), Slug: slug, Content: content,
		IsNew: isNew, Action: action, FormError: msg,
	}
	s.Render.Render(w, http.StatusBadRequest, "doc_edit", data)
}

// DocViewPage 是文档渲染页数据。
type DocViewPage struct {
	Page
	Slug    string
	Title   string
	Content string
}

// ViewDoc 渲染文档(markdown → 安全 HTML)。
func (s *Server) ViewDoc(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validSlug(slug) {
		http.NotFound(w, r)
		return
	}
	content, err := s.Content.Read(docPath(slug))
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, r, "读文档", err)
		return
	}
	data := DocViewPage{Page: s.page(r, docTitle(slug, content), "docs"), Slug: slug, Title: docTitle(slug, content), Content: content}
	s.Render.Render(w, http.StatusOK, "doc_view", data)
}

// EditDocForm 渲染编辑文档表单。
func (s *Server) EditDocForm(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validSlug(slug) {
		http.NotFound(w, r)
		return
	}
	content, err := s.Content.Read(docPath(slug))
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, r, "读文档", err)
		return
	}
	data := DocEditPage{
		Page: s.page(r, "编辑文档", "docs"), Slug: slug, Content: content,
		IsNew: false, Action: "/docs/" + slug,
	}
	s.Render.Render(w, http.StatusOK, "doc_edit", data)
}

// UpdateDoc 保存文档:写 git 并 commit。
func (s *Server) UpdateDoc(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validSlug(slug) {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	content := r.FormValue("content")
	if _, err := s.Content.Read(docPath(slug)); errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if _, err := s.Content.Commit(docPath(slug), []byte(content), "docs: 更新文档 "+slug); err != nil {
		s.serverError(w, r, "写入文档", err)
		return
	}
	redirectFlash(w, r, "/docs/"+slug, "ok", "已保存并提交到 git")
}
