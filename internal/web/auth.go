package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookie = "tally_session"
	sessionTTL    = 7 * 24 * time.Hour
)

// SessionStore 是 web 对会话存储的要求(在使用方定义接口)。
type SessionStore interface {
	CreateSession(ctx context.Context, token string, ttl time.Duration) error
	SessionExists(ctx context.Context, token string) (bool, error)
	DeleteSession(ctx context.Context, token string) error
}

// requireAuth 登录中间件:未登录请求重定向到 /login(INV-9)。
func (d Deps) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil || !d.validSession(r.Context(), c.Value) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (d Deps) validSession(ctx context.Context, token string) bool {
	if token == "" || d.Sessions == nil {
		return false
	}
	ok, err := d.Sessions.SessionExists(ctx, token)
	return err == nil && ok
}

// login 校验密码、签发会话。
func (d Deps) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(d.PasswordHash), []byte(r.FormValue("password"))) != nil {
		// 密码错误:回登录页。不区分「用户不存在/密码错」,单用户无需。
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}
	token, err := randomToken()
	if err != nil {
		http.Error(w, "内部错误", http.StatusInternalServerError)
		return
	}
	if err := d.Sessions.CreateSession(r.Context(), token, sessionTTL); err != nil {
		http.Error(w, "内部错误", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: int(sessionTTL.Seconds()),
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// logout 销毁会话。
func (d Deps) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && d.Sessions != nil {
		_ = d.Sessions.DeleteSession(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// randomToken 生成 32 字节随机 token(会话与 CSRF 共用)。
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
