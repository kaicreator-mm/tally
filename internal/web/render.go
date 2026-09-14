package web

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	assets "tally/web"
)

// fpPalette 是指纹条的调色板:取自 ui-spec 的语义色 + 一个中性灰。
// 颜色是固定常量,不从指纹字符串插值 —— 渲染结果只由「选哪个色」决定,天然安全。
var fpPalette = [...]string{
	"#9C3328", // --failed   氧化红
	"#2C4A61", // --accent   靛蓝
	"#A5731A", // --stale    赭石
	"#2C6146", // --verified 绿
	"#8A928A", // --ink-3    中性灰
}

// FingerprintBar 把一个指纹字符串确定性地渲染成 8 个色块的颜色序列。
//
// 纯函数:同一指纹在任何页面、任何时刻都得到同一条(同输入同输出)。
// 目的是让同一个反复出现的失败在扫列表时先被形状认出来,再去读十六进制
// (见 docs/06-ui-spec.md「签名元素:指纹条」)。
//
// 实现:对指纹做 sha256,取前 8 个字节,每字节对调色板长度取模选色。
// 取哈希而非直接解析十六进制,使任意形态的指纹(裸十六进制、sha256: 前缀、
// 含非十六进制字符)都能得到稳定且分布均匀的条。
func FingerprintBar(fp string) [8]string {
	sum := sha256.Sum256([]byte(fp))
	var bar [8]string
	for i := 0; i < 8; i++ {
		bar[i] = fpPalette[int(sum[i])%len(fpPalette)]
	}
	return bar
}

// fpShort 取指纹的短显示:去掉 sha256: 前缀,截前 6 位。纯展示用。
func fpShort(fp string) string {
	s := strings.TrimPrefix(fp, "sha256:")
	if len(s) > 6 {
		s = s[:6]
	}
	return s
}

// fpBarHTML 是指纹条的模板函数:输出 8 个色块的 <span class="fp-strip">。
// 颜色全部来自 fpPalette 常量,不含任何未转义的用户输入,故可安全作为 HTML 注入。
func fpBarHTML(fp string) template.HTML {
	bar := FingerprintBar(fp)
	var b strings.Builder
	b.WriteString(`<span class="fp-strip" role="img" aria-label="指纹 `)
	b.WriteString(template.HTMLEscapeString(fpShort(fp)))
	b.WriteString(`">`)
	for _, c := range bar {
		b.WriteString(`<i style="background:`)
		b.WriteString(c)
		b.WriteString(`"></i>`)
	}
	b.WriteString(`</span>`)
	return template.HTML(b.String())
}

// Renderer 持有解析好的页面模板。每个页面是 base + 该页 content 的一个克隆,
// 启动时一次解析,渲染期零解析开销。
type Renderer struct {
	pages map[string]*template.Template
}

// pageNames 是 T05a 落地的全部页面。T05b/c 新增页面时在此登记并在 web/templates 放同名 .tmpl。
var pageNames = []string{
	"index",
	"login",
	"projects", "project_form", "project_detail",
	"requirements", "requirement_edit",
	"tasks",
	"docs", "doc_view", "doc_edit",
	"wiki", "wiki_view", "wiki_edit",
	// T05b 执行引擎
	"environments", "deploys", "failures", "fingerprint",
	// T05c 追溯 + 知识 Wiki + 算出状态
	"knowledge", "diagnose", "coverage", "view_a", "view_b",
}

// funcMap 是注入全部模板的函数集。T05b/c 可在此追加 helper。
func funcMap() template.FuncMap {
	return template.FuncMap{
		"fpbar":    fpBarHTML,
		"fpshort":  fpShort,
		"markdown": renderMarkdown,
		"inc":      func(i int) int { return i + 1 }, // 排行序号:range 下标 → 人读的 1 起始
	}
}

// NewRenderer 从内嵌 FS 解析 base + 各页模板。
//
// 组合方式:base.tmpl 用 {{define "base"}}…{{template "content" .}}…{{end}} 定义外壳,
// 并给 content 一个空默认定义;每个页面 .tmpl 重新 {{define "content"}},克隆 base 后覆盖之。
func NewRenderer() (*Renderer, error) {
	base, err := template.New("base").Funcs(funcMap()).ParseFS(assets.TemplateFS, "templates/base.tmpl")
	if err != nil {
		return nil, fmt.Errorf("解析 base 模板: %w", err)
	}
	r := &Renderer{pages: make(map[string]*template.Template, len(pageNames))}
	for _, name := range pageNames {
		clone, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("克隆 base(%s): %w", name, err)
		}
		if _, err := clone.ParseFS(assets.TemplateFS, "templates/"+name+".tmpl"); err != nil {
			return nil, fmt.Errorf("解析页面 %s: %w", name, err)
		}
		r.pages[name] = clone
	}
	return r, nil
}

// Render 以给定 HTTP 状态码把 page 渲染到 w。data 传给模板的 base。
//
// 模板执行出错时:若尚未写出响应体则回 500;已开始写则只记日志(头已发出)。
func (r *Renderer) Render(w http.ResponseWriter, status int, page string, data any) {
	tmpl, ok := r.pages[page]
	if !ok {
		http.Error(w, "模板不存在: "+page, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		slog.Error("渲染模板", "page", page, "err", err)
	}
}
