package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type v15TestSessions struct{}

func (v15TestSessions) CreateSession(context.Context, string, time.Duration) error { return nil }
func (v15TestSessions) SessionExists(_ context.Context, token string) (bool, error) { return token == "valid", nil }
func (v15TestSessions) DeleteSession(context.Context, string) error { return nil }

func TestV15ControlPlaneRoutesStayBehindAuthAndCSRF(t *testing.T) {
	cp := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("v15-control-plane"))
	})
	r := NewRouter(Deps{Sessions: v15TestSessions{}, ControlPlane: cp})

	// Browser console is not public.
	req := httptest.NewRequest(http.MethodGet, "/v15/control-plane", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusSeeOther || res.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated console status=%d location=%q", res.Code, res.Header().Get("Location"))
	}

	// Authenticated GET is allowed and receives a CSRF cookie for subsequent
	// writes. This proves the v1.5 console uses the existing web security edge.
	req = httptest.NewRequest(http.MethodGet, "/v15/control-plane", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "valid"})
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("authenticated console status=%d body=%s", res.Code, res.Body.String())
	}
	var csrf *http.Cookie
	for _, c := range res.Result().Cookies() {
		if c.Name == csrfCookie {
			csrf = c
			break
		}
	}
	if csrf == nil || csrf.Value == "" {
		t.Fatal("authenticated console did not receive CSRF cookie")
	}

	// API writes require the same CSRF token; authentication alone is not enough.
	req = httptest.NewRequest(http.MethodPost, "/api/v15/projects/p/tasks/t/restart", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "valid"})
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("POST without CSRF status=%d, want 403", res.Code)
	}

	// Cookie + matching header crosses the security boundary and reaches the
	// mounted control-plane handler.
	req = httptest.NewRequest(http.MethodPost, "/api/v15/projects/p/tasks/t/restart", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "valid"})
	req.AddCookie(&http.Cookie{Name: csrfCookie, Value: csrf.Value})
	req.Header.Set(csrfHeader, csrf.Value)
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK || res.Body.String() != "v15-control-plane" {
		t.Fatalf("POST with CSRF status=%d body=%q", res.Code, res.Body.String())
	}
}

func TestV15ControlPlaneNavigationMatchesMountState(t *testing.T) {
	cp := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("v15-control-plane"))
	})

	for _, tc := range []struct {
		name              string
		controlPlane      http.Handler
		wantNav           bool
		wantRouteHTTPCode int
	}{
		{name: "disabled", controlPlane: nil, wantNav: false, wantRouteHTTPCode: http.StatusNotFound},
		{name: "enabled", controlPlane: cp, wantNav: true, wantRouteHTTPCode: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRouter(Deps{Sessions: v15TestSessions{}, ControlPlane: tc.controlPlane})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "valid"})
			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("root status=%d body=%s", res.Code, res.Body.String())
			}
			hasNav := strings.Contains(res.Body.String(), `href="/v15/control-plane"`)
			if hasNav != tc.wantNav {
				t.Fatalf("Control Plane mounted=%v nav visible=%v, want %v", tc.controlPlane != nil, hasNav, tc.wantNav)
			}

			req = httptest.NewRequest(http.MethodGet, "/v15/control-plane", nil)
			req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "valid"})
			res = httptest.NewRecorder()
			r.ServeHTTP(res, req)
			if res.Code != tc.wantRouteHTTPCode {
				t.Fatalf("Control Plane mounted=%v route status=%d, want %d", tc.controlPlane != nil, res.Code, tc.wantRouteHTTPCode)
			}
		})
	}
}
