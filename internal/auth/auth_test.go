// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAccounts(t *testing.T) {
	a, err := NewAccounts("admin-secret", "user-secret")
	if err != nil {
		t.Fatal(err)
	}
	if u, ok := a.Check("admin", "admin-secret"); !ok || !u.IsAdmin() {
		t.Errorf("admin: %+v, %v", u, ok)
	}
	if u, ok := a.Check("user", "user-secret"); !ok || u.IsAdmin() || u.Role != RoleUser {
		t.Errorf("user: %+v, %v", u, ok)
	}
	for _, c := range [][2]string{{"admin", "user-secret"}, {"user", ""}, {"root", "admin-secret"}, {"Admin", "admin-secret"}} {
		if _, ok := a.Check(c[0], c[1]); ok {
			t.Errorf("%s / %s must be refused", c[0], c[1])
		}
	}
}

// signIn returns the session cookie of a new session.
func signIn(t *testing.T, s *Sessions, u User, old *http.Cookie) *http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", nil)
	if old != nil {
		r.AddCookie(old)
	}
	s.SignIn(w, r, u)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %v", cookies)
	}
	return cookies[0]
}

func visit(s *Sessions, c *http.Cookie) (status int, user User) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/owner/list", nil)
	if c != nil {
		r.AddCookie(c)
	}
	s.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ = CurrentUser(r)
	})).ServeHTTP(w, r)
	return w.Code, user
}

func TestTheCookieIsProtected(t *testing.T) {
	c := signIn(t, NewSessions(time.Hour, true), User{Name: "user"}, nil)
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || len(c.Value) < 40 {
		t.Errorf("cookie = %+v", c)
	}
}

func TestRequireUser(t *testing.T) {
	s := NewSessions(time.Hour, false)
	if status, _ := visit(s, nil); status != http.StatusSeeOther {
		t.Errorf("without a session: %d", status)
	}
	if status, _ := visit(s, &http.Cookie{Name: CookieName, Value: "made-up"}); status != http.StatusSeeOther {
		t.Errorf("with an unknown session: %d", status)
	}
	c := signIn(t, s, User{Name: "admin", Role: RoleAdmin}, nil)
	if status, u := visit(s, c); status != http.StatusOK || u.Name != "admin" {
		t.Errorf("signed in: %d %+v", status, u)
	}
}

func TestSignInReplacesTheSession(t *testing.T) {
	s := NewSessions(time.Hour, false)
	first := signIn(t, s, User{Name: "user"}, nil)
	second := signIn(t, s, User{Name: "admin", Role: RoleAdmin}, first)
	if first.Value == second.Value {
		t.Fatal("a sign-in must get a new session id")
	}
	if status, _ := visit(s, first); status != http.StatusSeeOther {
		t.Error("the old session must be gone")
	}
}

func TestSignOut(t *testing.T) {
	s := NewSessions(time.Hour, false)
	c := signIn(t, s, User{Name: "user"}, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/logout", nil)
	r.AddCookie(c)
	s.SignOut(w, r)
	if status, _ := visit(s, c); status != http.StatusSeeOther {
		t.Error("the session must be gone after signing out")
	}
	if cleared := w.Result().Cookies(); len(cleared) != 1 || cleared[0].MaxAge != -1 {
		t.Errorf("the cookie must be cleared: %v", cleared)
	}
}

func TestIdleSessionsExpire(t *testing.T) {
	s := NewSessions(30*time.Minute, false)
	now := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	c := signIn(t, s, User{Name: "user"}, nil)

	now = now.Add(20 * time.Minute)
	if status, _ := visit(s, c); status != http.StatusOK {
		t.Fatal("20 minutes is within the timeout")
	}
	now = now.Add(29 * time.Minute) // 29 minutes after the last visit
	if status, _ := visit(s, c); status != http.StatusOK {
		t.Fatal("every visit restarts the timeout")
	}
	now = now.Add(31 * time.Minute)
	if status, _ := visit(s, c); status != http.StatusSeeOther {
		t.Error("the session must expire")
	}
}

func TestFlashIsShownOnce(t *testing.T) {
	s := NewSessions(time.Hour, false)
	c := signIn(t, s, User{Name: "user"}, nil)
	var first, second map[string]string
	handler := s.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set":
			s.Flash(r, "message", "Owner Ali was created")
		case "/first":
			first = s.TakeFlash(r)
		default:
			second = s.TakeFlash(r)
		}
	}))
	for _, path := range []string{"/set", "/first", "/second"} {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(c)
		handler.ServeHTTP(httptest.NewRecorder(), r)
	}
	if first["message"] != "Owner Ali was created" || len(second) != 0 {
		t.Errorf("first %v, second %v", first, second)
	}
}
