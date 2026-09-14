package arch

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"tally/internal/web"
)

// INV-9:所有产生副作用的路由(非 GET/HEAD/OPTIONS)且非公开的,必须过登录中间件。
// 行为式检查:未登录发起写请求必须被拒(非 2xx),而不是被正常处理。
func TestWriteRoutesRequireAuth(t *testing.T) {
	router := web.NewRouter(web.Deps{}) // 空依赖即可:鉴权中间件在 handler 之前就拒绝
	srv := httptest.NewServer(router)
	defer srv.Close()

	// 不跟随重定向:requireAuth 返回 303→/login,我们要看到那个 303 而不是跟过去的 200
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	public := map[string]bool{"/login": true, "/healthz": true}
	var bad []string
	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			return nil
		}
		if public[route] || strings.HasPrefix(route, "/static") {
			return nil
		}
		req, _ := http.NewRequest(method, srv.URL+route, strings.NewReader(""))
		resp, err := client.Do(req)
		if err != nil {
			bad = append(bad, method+" "+route+" (请求出错: "+err.Error()+")")
		} else {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				bad = append(bad, method+" "+route+" → "+strconv.Itoa(resp.StatusCode)+"(未要求登录)")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("遍历路由: %v", err)
	}
	if len(bad) > 0 {
		t.Errorf("存在未要求登录的写路由(INV-9):\n%s", strings.Join(bad, "\n"))
	}
}
