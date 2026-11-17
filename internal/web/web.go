// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

// Package web serves the pages: owners and pets with search, create, edit and delete, behind a login.
// Everybody who is signed in may read, create and edit; only admins may delete.
package web

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ergulcu/vetpet/internal/auth"
	"github.com/ergulcu/vetpet/internal/store"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

type ownerForm struct {
	ID                 int64
	Name, Email, Phone string
}

type petForm struct {
	ID                                          int64
	OwnerID, Name, Type, Kind, Age, Description string
}

// view is what every page gets.
type view struct {
	Title, Active  string
	Narrow         bool
	User           auth.User
	Message, Error string

	Search string
	Owners []store.Owner
	Pets   []store.Pet
	Owner  ownerForm
	Pet    petForm
	Errors map[string]string

	LoginFailed, LoggedOut bool

	Status             int
	StatusText, Detail string
}

type inputField struct {
	Name, Label, Type, Value, Error, Placeholder string
}

var funcs = template.FuncMap{
	"field": func(name, label, typ, value string, errs map[string]string) inputField {
		return inputField{Name: name, Label: label, Type: typ, Value: value, Error: errs[name]}
	},
	"placeholder": func(f inputField, p string) inputField {
		f.Placeholder = p
		return f
	},
	"age": func(age *int) string {
		if age == nil {
			return ""
		}
		return strconv.Itoa(*age)
	},
}

// Server holds what the handlers need.
type Server struct {
	store    *store.Store
	accounts *auth.Accounts
	sessions *auth.Sessions
	log      *slog.Logger
	pages    map[string]*template.Template
}

// New returns the handler of the application.
func New(st *store.Store, accounts *auth.Accounts, sessions *auth.Sessions, log *slog.Logger) (http.Handler, error) {
	s := &Server{store: st, accounts: accounts, sessions: sessions, log: log, pages: map[string]*template.Template{}}
	for _, page := range []string{"login", "owner_list", "owner_form", "pet_list", "pet_form", "error"} {
		t, err := template.New(page).Funcs(funcs).ParseFS(templateFS, "templates/layout.html", "templates/"+page+".html")
		if err != nil {
			return nil, err
		}
		s.pages[page] = t
	}
	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}

	app := http.NewServeMux()
	app.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/owner/list", http.StatusSeeOther)
	})
	app.HandleFunc("POST /logout", s.logout)
	app.HandleFunc("GET /owner/list", s.ownerList)
	app.HandleFunc("GET /owner/create", s.ownerCreateForm)
	app.HandleFunc("POST /owner/create", s.ownerSave)
	app.HandleFunc("GET /owner/update/{id}", s.ownerUpdateForm)
	app.HandleFunc("POST /owner/update/{id}", s.ownerSave)
	app.HandleFunc("POST /owner/delete/{id}", s.admin(s.ownerDelete))
	app.HandleFunc("GET /pet/list", s.petList)
	app.HandleFunc("GET /pet/create", s.petCreateForm)
	app.HandleFunc("POST /pet/create", s.petSave)
	app.HandleFunc("GET /pet/update/{id}", s.petUpdateForm)
	app.HandleFunc("POST /pet/update/{id}", s.petSave)
	app.HandleFunc("POST /pet/delete/{id}", s.admin(s.petDelete))
	app.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.fail(w, r, http.StatusNotFound, "There is no page at "+r.URL.Path+".")
	})

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Ping(r.Context()); err != nil {
			http.Error(w, "DOWN", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "UP")
	})
	mux.Handle("/", sessions.RequireUser(app))

	// Refuses form posts that another site makes the browser send (CSRF), using Sec-Fetch-Site and Origin.
	return http.NewCrossOriginProtection().Handler(securityHeaders(mux)), nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// ---- login ----

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	s.render(w, r, http.StatusOK, "login", &view{Title: "Log in", LoginFailed: q.Has("error"), LoggedOut: q.Has("logout")})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	user, ok := s.accounts.Check(r.PostFormValue("username"), r.PostFormValue("password"))
	if !ok {
		s.log.Info("failed login", "username", r.PostFormValue("username"), "remote", r.RemoteAddr)
		http.Redirect(w, r, "/login?error", http.StatusSeeOther)
		return
	}
	s.sessions.SignIn(w, r, user)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	s.sessions.SignOut(w, r)
	http.Redirect(w, r, "/login?logout", http.StatusSeeOther)
}

// admin lets only admins through; others get the access denied page.
func (s *Server) admin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u, _ := auth.CurrentUser(r); !u.IsAdmin() {
			s.fail(w, r, http.StatusForbidden, "Your account is not allowed to do this. Only administrators can delete records.")
			return
		}
		next(w, r)
	}
}

// ---- owners ----

func (s *Server) ownerList(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("name")
	owners, err := s.store.Owners(r.Context(), search)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "owner_list", &view{Title: "Owners", Active: "owners", Search: search, Owners: owners})
}

func (s *Server) ownerCreateForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "owner_form", &view{Title: "New owner", Active: "owners", Narrow: true})
}

func (s *Server) ownerUpdateForm(w http.ResponseWriter, r *http.Request) {
	o, ok := s.findOwner(w, r)
	if !ok {
		return
	}
	s.render(w, r, http.StatusOK, "owner_form", &view{Title: "Edit owner", Active: "owners", Narrow: true,
		Owner: ownerForm{ID: o.ID, Name: o.Name, Email: o.Email, Phone: o.Phone}})
}

// ownerSave creates (/owner/create) or updates (/owner/update/{id}).
func (s *Server) ownerSave(w http.ResponseWriter, r *http.Request) {
	var id int64
	title := "New owner"
	if r.PathValue("id") != "" {
		o, ok := s.findOwner(w, r)
		if !ok {
			return
		}
		id, title = o.ID, "Edit owner"
	}
	f := ownerForm{ID: id, Name: strings.TrimSpace(r.PostFormValue("name")),
		Email: strings.TrimSpace(r.PostFormValue("email")), Phone: strings.TrimSpace(r.PostFormValue("phone"))}
	errs := map[string]string{}
	required(errs, "name", f.Name, "Name is mandatory")
	maxLength(errs, "name", f.Name, 100)
	if f.Email != "" {
		if a, err := mail.ParseAddress(f.Email); err != nil || a.Address != f.Email {
			errs["email"] = "Email is not valid"
		}
	}
	maxLength(errs, "email", f.Email, 200)
	maxLength(errs, "phone", f.Phone, 30)
	if len(errs) > 0 {
		s.render(w, r, http.StatusOK, "owner_form", &view{Title: title, Active: "owners", Narrow: true, Owner: f, Errors: errs})
		return
	}
	o := store.Owner{ID: f.ID, Name: f.Name, Email: f.Email, Phone: f.Phone}
	if err := s.store.SaveOwner(r.Context(), &o); err != nil {
		s.internal(w, r, err)
		return
	}
	s.sessions.Flash(r, "message", fmt.Sprintf("Owner %s was %s", o.Name, done(id)))
	http.Redirect(w, r, "/owner/list", http.StatusSeeOther)
}

func (s *Server) ownerDelete(w http.ResponseWriter, r *http.Request) {
	o, ok := s.findOwner(w, r)
	if !ok {
		return
	}
	var hasPets *store.HasPetsError
	switch err := s.store.DeleteOwner(r.Context(), o.ID); {
	case errors.As(err, &hasPets):
		s.sessions.Flash(r, "error", fmt.Sprintf("%s still has %d pet(s); delete or move them first", o.Name, hasPets.Pets))
	case err != nil:
		s.internal(w, r, err)
		return
	default:
		s.sessions.Flash(r, "message", "Owner "+o.Name+" was deleted")
	}
	http.Redirect(w, r, "/owner/list", http.StatusSeeOther)
}

func (s *Server) findOwner(w http.ResponseWriter, r *http.Request) (store.Owner, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.fail(w, r, http.StatusNotFound, "Owner "+r.PathValue("id")+" not found")
		return store.Owner{}, false
	}
	o, err := s.store.Owner(r.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		s.fail(w, r, http.StatusNotFound, fmt.Sprintf("Owner %d not found", id))
		return o, false
	case err != nil:
		s.internal(w, r, err)
		return o, false
	}
	return o, true
}

// ---- pets ----

func (s *Server) petList(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("name")
	pets, err := s.store.Pets(r.Context(), search)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "pet_list", &view{Title: "Pets", Active: "pets", Search: search, Pets: pets})
}

func (s *Server) petCreateForm(w http.ResponseWriter, r *http.Request) {
	s.petForm(w, r, "New pet", petForm{}, nil)
}

func (s *Server) petUpdateForm(w http.ResponseWriter, r *http.Request) {
	p, ok := s.findPet(w, r)
	if !ok {
		return
	}
	f := petForm{ID: p.ID, OwnerID: strconv.FormatInt(p.OwnerID, 10), Name: p.Name, Type: p.Type, Kind: p.Kind,
		Description: p.Description}
	if p.Age != nil {
		f.Age = strconv.Itoa(*p.Age)
	}
	s.petForm(w, r, "Edit pet", f, nil)
}

// petForm shows the form with the owners for the drop-down.
func (s *Server) petForm(w http.ResponseWriter, r *http.Request, title string, f petForm, errs map[string]string) {
	owners, err := s.store.Owners(r.Context(), "")
	if err != nil {
		s.internal(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "pet_form", &view{Title: title, Active: "pets", Narrow: true, Pet: f, Owners: owners, Errors: errs})
}

// petSave creates (/pet/create) or updates (/pet/update/{id}).
func (s *Server) petSave(w http.ResponseWriter, r *http.Request) {
	var id int64
	title := "New pet"
	if r.PathValue("id") != "" {
		p, ok := s.findPet(w, r)
		if !ok {
			return
		}
		id, title = p.ID, "Edit pet"
	}
	field := func(name string) string { return strings.TrimSpace(r.PostFormValue(name)) }
	f := petForm{ID: id, OwnerID: field("owner"), Name: field("name"), Type: field("type"), Kind: field("kind"),
		Age: field("age"), Description: field("description")}
	errs := map[string]string{}
	p := store.Pet{ID: id, Name: f.Name, Type: f.Type, Kind: f.Kind, Description: f.Description}

	ownerID, err := strconv.ParseInt(f.OwnerID, 10, 64)
	if err == nil {
		_, err = s.store.Owner(r.Context(), ownerID)
	}
	if err != nil {
		errs["owner"] = "Owner is mandatory"
	}
	p.OwnerID = ownerID
	required(errs, "name", f.Name, "Name is mandatory")
	maxLength(errs, "name", f.Name, 100)
	required(errs, "type", f.Type, "Type is mandatory")
	maxLength(errs, "type", f.Type, 50)
	required(errs, "kind", f.Kind, "Kind is mandatory")
	maxLength(errs, "kind", f.Kind, 50)
	if f.Age != "" {
		switch age, err := strconv.Atoi(f.Age); {
		case err != nil:
			errs["age"] = "Age must be a whole number"
		case age < 0:
			errs["age"] = "Age cannot be negative"
		case age > 100:
			errs["age"] = "Age is not realistic"
		default:
			p.Age = &age
		}
	}
	maxLength(errs, "description", f.Description, 500)
	if len(errs) > 0 {
		s.petForm(w, r, title, f, errs)
		return
	}
	if err := s.store.SavePet(r.Context(), &p); err != nil {
		s.internal(w, r, err)
		return
	}
	s.sessions.Flash(r, "message", fmt.Sprintf("Pet %s was %s", p.Name, done(id)))
	http.Redirect(w, r, "/pet/list", http.StatusSeeOther)
}

func (s *Server) petDelete(w http.ResponseWriter, r *http.Request) {
	p, ok := s.findPet(w, r)
	if !ok {
		return
	}
	if err := s.store.DeletePet(r.Context(), p.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
		s.internal(w, r, err)
		return
	}
	s.sessions.Flash(r, "message", "Pet "+p.Name+" was deleted")
	http.Redirect(w, r, "/pet/list", http.StatusSeeOther)
}

func (s *Server) findPet(w http.ResponseWriter, r *http.Request) (store.Pet, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.fail(w, r, http.StatusNotFound, "Pet "+r.PathValue("id")+" not found")
		return store.Pet{}, false
	}
	p, err := s.store.Pet(r.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		s.fail(w, r, http.StatusNotFound, fmt.Sprintf("Pet %d not found", id))
		return p, false
	case err != nil:
		s.internal(w, r, err)
		return p, false
	}
	return p, true
}

// ---- helpers ----

func done(id int64) string {
	if id == 0 {
		return "created"
	}
	return "updated"
}

func required(errs map[string]string, field, value, message string) {
	if value == "" {
		errs[field] = message
	}
}

func maxLength(errs map[string]string, field, value string, max int) {
	if _, set := errs[field]; !set && utf8.RuneCountInString(value) > max {
		errs[field] = fmt.Sprintf("At most %d characters", max)
	}
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, page string, v *view) {
	if u, ok := auth.CurrentUser(r); ok {
		v.User = u
		flash := s.sessions.TakeFlash(r)
		v.Message, v.Error = flash["message"], flash["error"]
	}
	var buf bytes.Buffer
	if err := s.pages[page].ExecuteTemplate(&buf, "layout", v); err != nil {
		s.log.Error("render", "page", page, "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w) //nolint:errcheck
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, status int, detail string) {
	title := http.StatusText(status)
	if status == http.StatusForbidden {
		title = "Access denied"
	}
	s.render(w, r, status, "error", &view{Title: title, Status: status, StatusText: http.StatusText(status), Detail: detail})
}

func (s *Server) internal(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	s.fail(w, r, http.StatusInternalServerError, "Something went wrong.")
}
