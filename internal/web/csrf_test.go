package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestCSRFBlocksMissingToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.AddCookie(&http.Cookie{Name: csrfCookie, Value: "abc"})
	rr := httptest.NewRecorder()
	csrfMiddleware(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("缺 token 应 403,实际 %d", rr.Code)
	}
}

func TestCSRFBlocksMismatchedToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.AddCookie(&http.Cookie{Name: csrfCookie, Value: "abc"})
	req.Header.Set(csrfHeader, "xyz")
	rr := httptest.NewRecorder()
	csrfMiddleware(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("token 不一致应 403,实际 %d", rr.Code)
	}
}

func TestCSRFAllowsMatchingToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.AddCookie(&http.Cookie{Name: csrfCookie, Value: "abc"})
	req.Header.Set(csrfHeader, "abc")
	rr := httptest.NewRecorder()
	csrfMiddleware(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("token 一致应放行,实际 %d", rr.Code)
	}
}

func TestCSRFAllowsGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rr := httptest.NewRecorder()
	csrfMiddleware(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("GET 应直接放行,实际 %d", rr.Code)
	}
}
