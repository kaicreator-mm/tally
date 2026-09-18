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
func (v15TestSessions) SessionExists(_ context.Context, token string) (bool, error) {
	return token == "valid", nil
}
func (v15TestSessions) DeleteSession(context.Context, string) error { return nil }

func TestV15ControlPlaneNavigationTracksOuterRouteMount(t *testing.T) {
	cp := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("v15-control-plane"))
	})

	cases := []struct {
		name             string
		controlPlane     http.Handler
		wantNav          bool
		wantControlPlane int
	}{
		{name: "disabled", controlPlane: nil, wantNav: false, wantControlPlane: http.StatusNotFound},
		{name: "enabled", controlPlane: cp, wantNav: true, wantControlPlane: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRouter(Deps{Sessions: v15TestSessions{}, ControlPlane: tc.controlPlane})

			// /tasks is an existing protected page whose shared base template owns
			// the engineering navigation. Its nav must project the exact same
			// ControlPlane mount decision made by NewRouter.
			req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "valid"})
			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("tasks status=%d body=%s", res.Code, res.Body.String())
			}
			body := res.Body.String()
			hasControlPlaneNav := strings.Contains(body, `href="/v15/control-plane"`)
			if hasControlPlaneNav != tc.wantNav {
				t.Fatalf("ControlPlane nav present=%v, want %v", hasControlPlaneNav, tc.wantNav)
			}
			for _, sibling := range []string{
				`href="/v14/launch"`,
				`href="/v13/tasks"`,
				`href="/v13/runs"`,
				`href="/v14/attention"`,
				`href="/v14/rules"`,
				`href="/v14/integration"`,
			} {
				if !strings.Contains(body, sibling) {
					t.Errorf("ControlPlane mount decision changed sibling engineering nav %s", sibling)
				}
			}
			if tc.wantNav && !strings.Contains(body, ">工程控制台</a>") {
				t.Error("mounted ControlPlane nav lost its existing label")
			}

			// The route observation and the navigation observation must agree.
			req = httptest.NewRequest(http.MethodGet, "/v15/control-plane", nil)
			req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "valid"})
			res = httptest.NewRecorder()
			r.ServeHTTP(res, req)
			if res.Code != tc.wantControlPlane {
				t.Fatalf("ControlPlane route status=%d, want %d", res.Code, tc.wantControlPlane)
			}
		})
	}
}

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
