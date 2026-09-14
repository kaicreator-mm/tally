package project

import (
	"reflect"
	"testing"
)

// 纯解析逻辑的单元测试(不碰 git / 数据库)。集成测试见 *_integration_test.go(需 Postgres)。

func TestRefsFromMessage(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want []string
	}{
		{
			name: "标准 trailer",
			msg:  "feat: 做了东西\n\nTask: T04\nRefs: REQ-12",
			want: []string{"REQ-12"},
		},
		{
			name: "一行多个(逗号)",
			msg:  "feat: x\n\nRefs: REQ-12, REQ-7",
			want: []string{"REQ-12", "REQ-7"},
		},
		{
			name: "一行多个(空白)",
			msg:  "feat: x\n\nRefs: REQ-12 REQ-7",
			want: []string{"REQ-12", "REQ-7"},
		},
		{
			name: "多行 Refs",
			msg:  "feat: x\n\nRefs: REQ-1\nRefs: REQ-2",
			want: []string{"REQ-1", "REQ-2"},
		},
		{
			name: "子条款",
			msg:  "feat: x\n\nRefs: REQ-12.2",
			want: []string{"REQ-12.2"},
		},
		{
			name: "去首尾标点",
			msg:  "feat: x\n\nRefs: REQ-12.",
			want: []string{"REQ-12"},
		},
		{
			name: "键大小写不敏感",
			msg:  "feat: x\n\nrefs: REQ-12",
			want: []string{"REQ-12"},
		},
		{
			name: "无 Refs",
			msg:  "feat: x\n\nTask: T04",
			want: nil,
		},
		{
			name: "正文里的 Refs 不算(只看最后一段)",
			msg:  "Refs: REQ-99 是上一段的正文\n\nfeat: 实际提交无挂钩",
			want: nil,
		},
		{
			name: "非条款编号被过滤",
			msg:  "feat: x\n\nRefs: REQ-12, FOO, #12",
			want: []string{"REQ-12"},
		},
		{
			name: "去重",
			msg:  "feat: x\n\nRefs: REQ-12, REQ-12",
			want: []string{"REQ-12"},
		},
		{
			name: "单段消息(无空行)",
			msg:  "Refs: REQ-12",
			want: []string{"REQ-12"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RefsFromMessage(tc.msg); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("RefsFromMessage(%q) = %v, want %v", tc.msg, got, tc.want)
			}
		})
	}
}

func TestCoversFromSource(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "Go 注释单个",
			src:  "// @covers REQ-12.2\nfunc TestX(t *testing.T) {}",
			want: []string{"REQ-12.2"},
		},
		{
			name: "一行多个",
			src:  "# @covers REQ-1, REQ-2",
			want: []string{"REQ-1", "REQ-2"},
		},
		{
			name: "多处注解去重",
			src:  "// @covers REQ-1\nfunc a(){}\n// @covers REQ-1\nfunc b(){}\n// @covers REQ-3",
			want: []string{"REQ-1", "REQ-3"},
		},
		{
			name: "块注释尾部标点",
			src:  "/* @covers REQ-12 */",
			want: []string{"REQ-12"},
		},
		{
			name: "@coversFoo 不误匹配",
			src:  "// @coversSomething REQ-1",
			want: nil,
		},
		{
			name: "无注解",
			src:  "func TestX(t *testing.T) {}",
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CoversFromSource(tc.src); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("CoversFromSource(%q) = %v, want %v", tc.src, got, tc.want)
			}
		})
	}
}
