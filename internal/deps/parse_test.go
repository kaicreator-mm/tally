package deps

import (
	"testing"
)

func TestParseGoMod(t *testing.T) {
	src := []byte(`module example.com/demo

go 1.23

require (
	github.com/jackc/pgx/v5 v5.5.1
	github.com/go-chi/chi/v5 v5.0.12 // indirect
)

require golang.org/x/sync v0.10.0

replace old.example.com/lib => new.example.com/lib v1.0.0
`)
	got, err := ParseGoMod(src)
	if err != nil {
		t.Fatalf("ParseGoMod: %v", err)
	}
	want := map[string]string{
		"github.com/jackc/pgx/v5":  "v5.5.1",
		"github.com/go-chi/chi/v5": "v5.0.12",
		"golang.org/x/sync":        "v0.10.0",
	}
	if len(got) != len(want) {
		t.Fatalf("条目数 = %d,想要 %d:%v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q,想要 %q", k, got[k], v)
		}
	}
}

func TestParseGoModEmpty(t *testing.T) {
	got, err := ParseGoMod([]byte("module example.com/none\n"))
	if err != nil {
		t.Fatalf("ParseGoMod: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("空 require 应得空 map,得到 %v", got)
	}
}

func TestParsePackageLock(t *testing.T) {
	src := []byte(`{
  "name": "demo",
  "lockfileVersion": 3,
  "packages": {
    "": {"name": "demo", "version": "0.0.1"},
    "node_modules/left-pad": {"version": "1.3.0"},
    "node_modules/a/node_modules/left-pad": {"version": "1.1.0"}
  }
}`)
	got, err := ParsePackageLock(src)
	if err != nil {
		t.Fatalf("ParsePackageLock: %v", err)
	}
	// 嵌套同名依赖以深处为准(D-a6)。
	if got["left-pad"] != "1.1.0" {
		t.Errorf("left-pad = %q,想要 1.1.0(嵌套更深者)", got["left-pad"])
	}
	if _, ok := got["demo"]; ok {
		t.Errorf("根条目(空键)不应算依赖:%v", got)
	}
	if len(got) != 1 {
		t.Errorf("条目数 = %d,想要 1:%v", len(got), got)
	}
}

func TestParsePackageLockNoPackages(t *testing.T) {
	if _, err := ParsePackageLock([]byte(`{"dependencies": {}}`)); err == nil {
		t.Error("缺少 packages 段应报错(不支持 lockfileVersion 1)")
	}
}

func TestDiff(t *testing.T) {
	before := map[string]string{"a": "1.0.0", "b": "2.0.0", "c": "3.0.0"}
	after := map[string]string{"a": "1.1.0", "c": "3.0.0", "d": "4.0.0"}
	got := Diff(before, after)
	want := []Change{
		{Package: "a", From: "1.0.0", To: "1.1.0"}, // 改
		{Package: "b", From: "2.0.0", To: ""},      // 删
		{Package: "d", From: "", To: "4.0.0"},      // 增
	}
	if len(got) != len(want) {
		t.Fatalf("变更数 = %d,想要 %d:%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 条 = %+v,想要 %+v", i, got[i], want[i])
		}
	}
}

func TestDiffSame(t *testing.T) {
	m := map[string]string{"a": "1.0.0"}
	if got := Diff(m, m); len(got) != 0 {
		t.Errorf("相同映射应无变更:%v", got)
	}
}
