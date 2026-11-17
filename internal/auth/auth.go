// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

// Package auth signs users in with a form and keeps them in a server side session, as Spring Security's
// form login did. Sessions live in memory: a restart signs everybody out.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Role decides what a user may do: only admins may delete.
type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

// User is a signed-in user.
type User struct {
	Name string
	Role Role
}

func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

type account struct {
	hash []byte
	role Role
}

// Accounts are the users who can sign in, with hashed passwords.
type Accounts struct {
	accounts map[string]account
	// dummy is checked for unknown names, so that they take as long as a wrong password.
	dummy []byte
}

// NewAccounts creates the two demo accounts: admin may delete, user may not.
func NewAccounts(adminPassword, userPassword string) (*Accounts, error) {
	a := &Accounts{accounts: map[string]account{}}
	for name, acc := range map[string]struct {
		password string
		role     Role
	}{"admin": {adminPassword, RoleAdmin}, "user": {userPassword, RoleUser}} {
		hash, err := bcrypt.GenerateFromPassword([]byte(acc.password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		a.accounts[name] = account{hash: hash, role: acc.role}
	}
	a.dummy, _ = bcrypt.GenerateFromPassword([]byte("dummy password"), bcrypt.DefaultCost)
	return a, nil
}

// Check returns the user when the password is right.
func (a *Accounts) Check(name, password string) (User, bool) {
	acc, ok := a.accounts[name]
	if !ok {
		bcrypt.CompareHashAndPassword(a.dummy, []byte(password)) //nolint:errcheck
		return User{}, false
	}
	if bcrypt.CompareHashAndPassword(acc.hash, []byte(password)) != nil {
		return User{}, false
	}
	return User{Name: name, Role: acc.role}, true
}

// CookieName is the session cookie.
const CookieName = "vetpet_session"

type session struct {
	user     User
	lastSeen time.Time
	flash    map[string]string
}

// Sessions keeps the signed-in users. It is safe for concurrent use.
type Sessions struct {
	mu       sync.Mutex
	sessions map[string]*session
	// IdleTimeout signs a user out who has not been seen for this long.
	IdleTimeout time.Duration
	// Secure marks the cookie for HTTPS only.
	Secure bool
	now    func() time.Time
}

func NewSessions(idleTimeout time.Duration, secure bool) *Sessions {
	return &Sessions{sessions: map[string]*session{}, IdleTimeout: idleTimeout, Secure: secure, now: time.Now}
}

// SignIn starts a new session for the user. A new id is used every time, so that an id planted before
// the sign-in (session fixation) is worthless.
func (s *Sessions) SignIn(w http.ResponseWriter, r *http.Request, u User) {
	if c, err := r.Cookie(CookieName); err == nil {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	b := make([]byte, 32)
	rand.Read(b) //nolint:errcheck // never fails
	id := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	s.sessions[id] = &session{user: u, lastSeen: s.now()}
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: id, Path: "/", HttpOnly: true, Secure: s.Secure,
		SameSite: http.SameSiteLaxMode})
}

// SignOut ends the session.
func (s *Sessions) SignOut(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		Secure: s.Secure, SameSite: http.SameSiteLaxMode})
}

func (s *Sessions) find(r *http.Request) *session {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for id, sess := range s.sessions { // drop expired sessions as we go
		if now.Sub(sess.lastSeen) > s.IdleTimeout {
			delete(s.sessions, id)
		}
	}
	sess := s.sessions[c.Value]
	if sess != nil {
		sess.lastSeen = now
	}
	return sess
}

type contextKey struct{}

// RequireUser lets signed-in users through and sends everyone else to the login page.
func (s *Sessions) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := s.find(r)
		if sess == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, sess)))
	})
}

// CurrentUser is the user of a request that passed RequireUser.
func CurrentUser(r *http.Request) (User, bool) {
	sess, ok := r.Context().Value(contextKey{}).(*session)
	if !ok {
		return User{}, false
	}
	return sess.user, true
}

// Flash keeps a message for the next page the user sees ("Owner Ali was created").
func (s *Sessions) Flash(r *http.Request, kind, message string) {
	sess, ok := r.Context().Value(contextKey{}).(*session)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess.flash == nil {
		sess.flash = map[string]string{}
	}
	sess.flash[kind] = message
}

// TakeFlash returns the kept messages and forgets them.
func (s *Sessions) TakeFlash(r *http.Request) map[string]string {
	sess, ok := r.Context().Value(contextKey{}).(*session)
	if !ok {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f := sess.flash
	sess.flash = nil
	return f
}
