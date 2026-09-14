package web

import (
	"context"
	"crypto/subtle"
	"net/http"

	"tally/internal/web/handlers"
)

const (
	csrfCookie = "tally_csrf"
	csrfHeader = "X-CSRF-Token"
	csrfField  = "csrf_token"
)

// csrfMiddleware 是双提交 token 防护:写请求必须带与 cookie 一致的 token,否则 403。
// 手写,不引依赖(INV-6)。GET/HEAD/OPTIONS 放行。
//
// 有效 token 还会写进请求 context(handlers.CSRFTokenKey),供表单渲染时填隐藏域 ——
// 首次访问时 cookie 刚由本中间件写进响应、请求里还没有,handler 只能从 context 拿。
func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(csrfCookie)
		if err != nil || cookie.Value == "" {
			token, terr := randomToken()
			if terr != nil {
				http.Error(w, "内部错误", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: csrfCookie, Value: token, Path: "/", SameSite: http.SameSiteLaxMode})
			cookie = &http.Cookie{Value: token}
		}
		// 让下游 handler 拿到当前有效 token(填表单隐藏域用)。
		r = r.WithContext(context.WithValue(r.Context(), handlers.CSRFTokenKey, cookie.Value))

		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		sent := r.Header.Get(csrfHeader)
		if sent == "" {
			_ = r.ParseForm()
			sent = r.FormValue(csrfField)
		}
		if sent == "" || subtle.ConstantTimeCompare([]byte(sent), []byte(cookie.Value)) != 1 {
			http.Error(w, "CSRF token 不一致", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
