package harvest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 验收:清单可配置、默认十来个;无 200+ 批处理路径。
func TestDefaultServicesSmall(t *testing.T) {
	n := len(DefaultServices)
	if n < 5 || n > 15 {
		t.Errorf("默认清单应约十来个, got %d", n)
	}
	if n >= 50 {
		t.Errorf("默认清单不得接近全量批处理(Coolify 有 362 个), got %d", n)
	}
	seen := map[string]bool{}
	for _, s := range DefaultServices {
		if seen[s.Name] {
			t.Errorf("清单重复: %s", s.Name)
		}
		seen[s.Name] = true
		if s.Name == "" || s.Source == "" {
			t.Errorf("清单项缺 name/source: %+v", s)
		}
	}
}

// 清单里的服务必须能被 Run 接受(来源名都有对应适配器可建)——
// 即每个 Source 都在 coolify/upstream/docs 之内,不存在未定义的来源。
func TestDefaultServicesSourcesKnown(t *testing.T) {
	known := map[string]bool{"coolify": true, "upstream": true, "docs": true}
	for _, s := range DefaultServices {
		if !known[s.Source] {
			t.Errorf("服务 %s 的来源 %q 无适配器", s.Name, s.Source)
		}
	}
}

// CoolifySource 只读取材料,URL 指向公开模板目录(不依赖运行时)。
func TestCoolifySourceFetch(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte("services:\n  minio:\n    image: x\n"))
	}))
	defer srv.Close()

	src := CoolifySource{BaseURL: srv.URL, Fetcher: HTTPFetcher{}}
	mat, err := src.Fetch(context.Background(), Service{Name: "minio", Source: "coolify", Ref: "minio"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !strings.HasSuffix(gotPath, "/minio.yaml") {
		t.Errorf("应取 <base>/minio.yaml: %s", gotPath)
	}
	if !strings.Contains(mat.Raw, "minio") {
		t.Errorf("应返回原始材料: %q", mat.Raw)
	}
	if src.Name() != "coolify" {
		t.Errorf("来源名: %s", src.Name())
	}
}

// 取材料失败(如 404)→ 返回错误。
func TestCoolifySourceFetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	src := CoolifySource{BaseURL: srv.URL, Fetcher: HTTPFetcher{}}
	if _, err := src.Fetch(context.Background(), Service{Ref: "nope"}); err == nil {
		t.Fatal("404 应返回错误")
	}
}

// HTTPSource(upstream/docs)按 URL 取材料。
func TestHTTPSourceFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("upstream compose"))
	}))
	defer srv.Close()
	src := HTTPSource{SourceName: "upstream", Fetcher: HTTPFetcher{}}
	mat, err := src.Fetch(context.Background(), Service{Name: "redis", Source: "upstream", Ref: srv.URL})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if mat.Raw != "upstream compose" {
		t.Errorf("应返回材料: %q", mat.Raw)
	}
	if src.Name() != "upstream" {
		t.Errorf("来源名: %s", src.Name())
	}
}

// StaticSource 缺材料 → 报错。
func TestStaticSourceMissing(t *testing.T) {
	src := StaticSource{SourceName: "coolify", Content: map[string]string{}}
	if _, err := src.Fetch(context.Background(), Service{Ref: "x"}); err == nil {
		t.Fatal("缺材料应报错")
	}
}
