package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Wiki 笔记层:git 里的 wiki/<slug>.md,可编辑,但界面一律标「笔记·未验证」(--stale 赭石)。
// 与「已验证修复库」(--verified 绿,只读,T05c)分目录、分色、分标,视觉上绝不雷同。

func wikiPath(slug string) string { return "wiki/" + slug + ".md" }

// NoteView 是笔记列表的一项。
type NoteView struct {
	Slug  string
	Title string
}

// WikiPage 是笔记列表页数据。
type WikiPage struct {
	Page
	Notes []NoteView
}

// ListWiki 列出全部笔记。
func (s *Server) ListWiki(w http.ResponseWriter, r *http.Request) {
	files, err := s.Content.List("wiki", ".md")
	if err != nil {
		s.serverError(w, r, "读笔记列表", err)
		return
	}
	data := WikiPage{Page: s.page(r, "Wiki 笔记", "wiki")}
	for _, f := range files {
		slug := strings.TrimSuffix(strings.TrimPrefix(f, "wiki/"), ".md")
		title := slug
		if content, err := s.Content.Read(f); err == nil {
			title = docTitle(slug, content)
		}
		data.Notes = append(data.Notes, NoteView{Slug: slug, Title: title})
	}
	s.Render.Render(w, http.StatusOK, "wiki", data)
}

// WikiEditPage 是笔记编辑页数据(新建/编辑共用)。
type WikiEditPage struct {
	Page
	Slug      string
	Content   string
	IsNew     bool
	Action    string
	FormError string
}

// NewWikiForm 渲染新建笔记表单。
func (s *Server) NewWikiForm(w http.ResponseWriter, r *http.Request) {
	data := WikiEditPage{Page: s.page(r, "新建笔记", "wiki"), IsNew: true, Action: "/wiki"}
	s.Render.Render(w, http.StatusOK, "wiki_edit", data)
}

// CreateWiki 新建笔记:写 git wiki/<slug>.md 并 commit。
func (s *Server) CreateWiki(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	slug := strings.TrimSpace(r.FormValue("slug"))
	content := r.FormValue("content")
	if !validSlug(slug) {
		s.renderWikiFormError(w, r, true, slug, content, "标识只能用小写字母、数字、连字符")
		return
	}
	if _, err := s.Content.Read(wikiPath(slug)); err == nil {
		s.renderWikiFormError(w, r, true, slug, content, "标识 "+slug+" 已存在")
		return
	} else if !errors.Is(err, ErrNotFound) {
		s.serverError(w, r, "检查笔记", err)
		return
	}
	if _, err := s.Content.Commit(wikiPath(slug), []byte(content), "docs: 新建笔记 "+slug); err != nil {
		s.serverError(w, r, "写入笔记", err)
		return
	}
	redirectFlash(w, r, "/wiki/"+slug, "ok", "笔记已写入 git(标「未验证」)")
}

func (s *Server) renderWikiFormError(w http.ResponseWriter, r *http.Request, isNew bool, slug, content, msg string) {
	action := "/wiki"
	if !isNew {
		action = "/wiki/" + slug
	}
	data := WikiEditPage{
		Page: s.page(r, "新建笔记", "wiki"), Slug: slug, Content: content,
		IsNew: isNew, Action: action, FormError: msg,
	}
	s.Render.Render(w, http.StatusBadRequest, "wiki_edit", data)
}

// WikiViewPage 是笔记渲染页数据。
type WikiViewPage struct {
	Page
	Slug    string
	Title   string
	Content string
}

// ViewWiki 渲染笔记(markdown → 安全 HTML,页面带「笔记·未验证」标)。
func (s *Server) ViewWiki(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validSlug(slug) {
		http.NotFound(w, r)
		return
	}
	content, err := s.Content.Read(wikiPath(slug))
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, r, "读笔记", err)
		return
	}
	title := docTitle(slug, content)
	data := WikiViewPage{Page: s.page(r, title, "wiki"), Slug: slug, Title: title, Content: content}
	s.Render.Render(w, http.StatusOK, "wiki_view", data)
}

// EditWikiForm 渲染编辑笔记表单。
func (s *Server) EditWikiForm(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !validSlug(slug) {
		http.NotFound(w, r)
		return
	}
	content, err := s.Content.Read(wikiPath(slug))
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, r, "读笔记", err)
		return
	}
	data := WikiEditPage{
		Page: s.page(r, "编辑笔记", "wiki"), Slug: slug, Content: content,
		IsNew: false, Action: "/wiki/" + slug,
	}
	s.Render.Render(w, http.StatusOK, "wiki_edit", data)
}

// UpdateWiki 保存笔记:写 git 并 commit。
func (s *Server) UpdateWiki(w http.ResponseWriter, r *http.Request) {
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
	if _, err := s.Content.Read(wikiPath(slug)); errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if _, err := s.Content.Commit(wikiPath(slug), []byte(content), "docs: 更新笔记 "+slug); err != nil {
		s.serverError(w, r, "写入笔记", err)
		return
	}
	redirectFlash(w, r, "/wiki/"+slug, "ok", "已保存并提交到 git")
}
