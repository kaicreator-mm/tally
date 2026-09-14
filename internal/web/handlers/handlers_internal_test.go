package handlers

import "testing"

// 纯逻辑单测:路径校验(防越界)与项目/文档的 markdown 解析。不碰网络与数据库。

func TestValidSlug(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"普通标识", "forge", true},
		{"带数字连字符", "my-proj-2", true},
		{"空", "", false},
		{"大写", "Forge", false},
		{"下划线", "my_proj", false},
		{"点", "a.b", false},
		{"路径分隔符", "a/b", false},
		{"上级目录", "..", false},
		{"含 .. 的穿越", "../etc", false},
		{"空格", "a b", false},
		{"过长", string(make([]byte, 65)), false},
	}
	for _, tt := range tests {
		if got := validSlug(tt.in); got != tt.want {
			t.Errorf("%s: validSlug(%q) = %v, 想要 %v", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestValidFileName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"正常 md", "multi-tenant.md", true},
		{"带下划线", "req_1.md", true},
		{"非 md 后缀", "req.txt", false},
		{"无后缀", "req", false},
		{"含分隔符", "../x.md", false},
		{"空", "", false},
		{"仅后缀", ".md", true},
	}
	for _, tt := range tests {
		if got := validFileName(tt.in); got != tt.want {
			t.Errorf("%s: validFileName(%q) = %v, 想要 %v", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestParseProject(t *testing.T) {
	content := "# Forge\n\n一个构建工具。\n\n其它正文。\n"
	p := parseProject("forge", content)
	if p.Slug != "forge" || p.Name != "Forge" || p.Description != "一个构建工具。" {
		t.Errorf("解析项目不对: %+v", p)
	}
}

func TestParseProjectNoHeading(t *testing.T) {
	p := parseProject("forge", "没有标题的正文\n")
	if p.Name != "forge" { // 无 H1 时名称落回 slug
		t.Errorf("无标题时名称应落回 slug,得到 %q", p.Name)
	}
}

func TestSerializeProjectRoundTrip(t *testing.T) {
	content := serializeProject("Forge", "一个构建工具。")
	p := parseProject("forge", content)
	if p.Name != "Forge" || p.Description != "一个构建工具。" {
		t.Errorf("序列化-解析往返不一致: %+v(内容 %q)", p, content)
	}
}

func TestDocTitle(t *testing.T) {
	if got := docTitle("arch", "# 架构说明\n\n正文"); got != "架构说明" {
		t.Errorf("docTitle = %q, 想要 架构说明", got)
	}
	if got := docTitle("arch", "没有标题"); got != "arch" {
		t.Errorf("无标题时 docTitle 应落回 slug,得到 %q", got)
	}
}

func TestShortHash(t *testing.T) {
	if got := shortHash("sha256:abcdef123456"); got != "abcdef" {
		t.Errorf("shortHash = %q, 想要 abcdef", got)
	}
	if got := shortHash("abc"); got != "abc" {
		t.Errorf("短于 6 位应原样返回,得到 %q", got)
	}
}
