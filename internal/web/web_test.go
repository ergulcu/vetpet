// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ergulcu/vetpet/internal/auth"
	"github.com/ergulcu/vetpet/internal/store"
)

type app struct {
	srv   *httptest.Server
	store *store.Store
}

// accounts is created once: bcrypt is slow on purpose.
var accounts, _ = auth.NewAccounts("admin-pw", "user-pw")

func newApp(t *testing.T) *app {
	t.Helper()
	st, err := store.Open(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	h, err := New(st, accounts, auth.NewSessions(time.Hour, false), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &app{srv: srv, store: st}
}

type browser struct {
	t      *testing.T
	app    *app
	client *http.Client
}

func (a *app) browser(t *testing.T) *browser {
	jar, _ := cookiejar.New(nil)
	return &browser{t: t, app: a, client: &http.Client{Jar: jar}}
}

// signedIn returns a browser signed in as the user.
func (a *app) signedIn(t *testing.T, name string) *browser {
	b := a.browser(t)
	if _, body := b.post("/login", url.Values{"username": {name}, "password": {name + "-pw"}}); !strings.Contains(body, "User: <strong>"+name) {
		t.Fatalf("sign in as %s failed: %s", name, body)
	}
	return b
}

type page struct {
	status int
	path   string
}

func (b *browser) do(req *http.Request) (page, string) {
	b.t.Helper()
	resp, err := b.client.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	p := resp.Request.URL.Path
	if resp.Request.URL.RawQuery != "" {
		p += "?" + resp.Request.URL.RawQuery
	}
	return page{resp.StatusCode, p}, string(body)
}

func (b *browser) get(path string) (page, string) {
	req, _ := http.NewRequest("GET", b.app.srv.URL+path, nil)
	return b.do(req)
}

func (b *browser) post(path string, form url.Values) (page, string) {
	req, _ := http.NewRequest("POST", b.app.srv.URL+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// What a browser sends for a form on the same site.
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	return b.do(req)
}

func contains(t *testing.T, body string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(body, w) {
			t.Errorf("the page does not contain %q", w)
		}
	}
}

func TestEveryPageNeedsALogin(t *testing.T) {
	a := newApp(t)
	b := a.browser(t)
	for _, path := range []string{"/", "/owner/list", "/pet/create", "/owner/update/1"} {
		if p, body := b.get(path); p.path != "/login" || !strings.Contains(body, "Log in") {
			t.Errorf("%s ended at %s", path, p.path)
		}
	}
	if p, _ := b.post("/owner/delete/1", nil); p.path != "/login" {
		t.Errorf("a post without login ended at %s", p.path)
	}
}

func TestLogin(t *testing.T) {
	a := newApp(t)
	b := a.browser(t)
	p, body := b.post("/login", url.Values{"username": {"user"}, "password": {"wrong"}})
	if p.path != "/login?error" {
		t.Errorf("wrong password ended at %s", p.path)
	}
	contains(t, body, "Invalid username or password.")

	b = a.signedIn(t, "user")
	p, body = b.post("/logout", nil)
	if p.path != "/login?logout" {
		t.Errorf("logout ended at %s", p.path)
	}
	contains(t, body, "You have been logged out.")
	if p, _ := b.get("/owner/list"); p.path != "/login" {
		t.Error("the session must be gone after logout")
	}
}

func TestOwners(t *testing.T) {
	a := newApp(t)
	b := a.signedIn(t, "user")

	p, body := b.post("/owner/create", url.Values{"name": {" "}, "email": {"not-an-address"}, "phone": {strings.Repeat("1", 31)}})
	if p.status != http.StatusOK || p.path != "/owner/create" {
		t.Errorf("invalid form: %+v", p)
	}
	contains(t, body, "Name is mandatory", "Email is not valid", "At most 30 characters", `value="not-an-address"`, "is-invalid")

	p, body = b.post("/owner/create", url.Values{"name": {"Şükrü Kaya"}, "email": {"sukru@example.com"}, "phone": {"0555 111 22 33"}})
	if p.path != "/owner/list" {
		t.Errorf("create ended at %s", p.path)
	}
	contains(t, body, "Owner Şükrü Kaya was created", "sukru@example.com", "0555 111 22 33")
	if _, body := b.get("/owner/list"); strings.Contains(body, "was created") {
		t.Error("a flash message is shown only once")
	}

	owners, _ := a.store.Owners(context.Background(), "")
	id := owners[0].ID
	_, body = b.get("/owner/update/" + itoa(id))
	contains(t, body, "Edit owner", `value="Şükrü Kaya"`)
	_, body = b.post("/owner/update/"+itoa(id), url.Values{"name": {"Şükrü Yılmaz"}})
	contains(t, body, "Owner Şükrü Yılmaz was updated")

	b.post("/owner/create", url.Values{"name": {"Ali"}})
	_, body = b.get("/owner/list?name=ŞÜK")
	contains(t, body, "Şükrü Yılmaz")
	if strings.Contains(body, "<td>Ali</td>") {
		t.Error("the search must leave Ali out")
	}
}

func TestOnlyAdminsDelete(t *testing.T) {
	a := newApp(t)
	o := store.Owner{Name: "Ali"}
	a.store.SaveOwner(context.Background(), &o) //nolint:errcheck

	user := a.signedIn(t, "user")
	_, body := user.get("/owner/list")
	if strings.Contains(body, "/owner/delete/") {
		t.Error("users must not see the delete button")
	}
	p, body := user.post("/owner/delete/"+itoa(o.ID), nil)
	if p.status != http.StatusForbidden {
		t.Errorf("status %d", p.status)
	}
	contains(t, body, "Access denied", "Only administrators can delete records.")
	if _, err := a.store.Owner(context.Background(), o.ID); err != nil {
		t.Error("the owner must still be there")
	}

	admin := a.signedIn(t, "admin")
	_, body = admin.get("/owner/list")
	contains(t, body, "/owner/delete/"+itoa(o.ID))
	p, body = admin.post("/owner/delete/"+itoa(o.ID), nil)
	if p.path != "/owner/list" {
		t.Errorf("delete ended at %s", p.path)
	}
	contains(t, body, "Owner Ali was deleted", "No owners found.")
}

func TestAnOwnerWithPetsIsNotDeleted(t *testing.T) {
	a := newApp(t)
	o := store.Owner{Name: "Ali"}
	a.store.SaveOwner(context.Background(), &o)                                                                 //nolint:errcheck
	a.store.SavePet(context.Background(), &store.Pet{OwnerID: o.ID, Name: "Tekir", Type: "Cat", Kind: "Tabby"}) //nolint:errcheck

	_, body := a.signedIn(t, "admin").post("/owner/delete/"+itoa(o.ID), nil)
	contains(t, body, "Ali still has 1 pet(s); delete or move them first", "<td>Ali</td>")
}

func TestPets(t *testing.T) {
	a := newApp(t)
	b := a.signedIn(t, "user")
	_, body := b.get("/pet/create")
	contains(t, body, "There are no owners yet.")

	o := store.Owner{Name: "Ali"}
	a.store.SaveOwner(context.Background(), &o) //nolint:errcheck

	_, body = b.post("/pet/create", url.Values{"name": {"Karabaş"}, "age": {"abc"}})
	contains(t, body, "Owner is mandatory", "Type is mandatory", "Kind is mandatory", "Age must be a whole number")
	_, body = b.post("/pet/create", url.Values{"owner": {"999"}, "name": {"Karabaş"}, "type": {"Dog"}, "kind": {"Kangal"}, "age": {"150"}})
	contains(t, body, "Owner is mandatory", "Age is not realistic")
	_, body = b.post("/pet/create", url.Values{"owner": {itoa(o.ID)}, "name": {"Karabaş"}, "type": {"Dog"}, "kind": {"Kangal"}, "age": {"-1"}})
	contains(t, body, "Age cannot be negative", `<option value="`+itoa(o.ID)+`" selected>Ali</option>`)

	p, body := b.post("/pet/create", url.Values{"owner": {itoa(o.ID)}, "name": {"Karabaş"}, "type": {"Dog"},
		"kind": {"Kangal"}, "age": {"4"}, "description": {"Guards the garden"}})
	if p.path != "/pet/list" {
		t.Errorf("create ended at %s", p.path)
	}
	contains(t, body, "Pet Karabaş was created", "<td>Kangal</td>", "<td>4</td>", "<td>Guards the garden</td>", "<td>Ali</td>")

	pets, _ := a.store.Pets(context.Background(), "")
	_, body = b.post("/pet/update/"+itoa(pets[0].ID), url.Values{"owner": {itoa(o.ID)}, "name": {"Karabaş"},
		"type": {"Dog"}, "kind": {"Kangal"}})
	contains(t, body, "Pet Karabaş was updated")
	if got, _ := a.store.Pet(context.Background(), pets[0].ID); got.Age != nil {
		t.Errorf("an empty age is unknown: %v", *got.Age)
	}

	p, _ = b.post("/pet/delete/"+itoa(pets[0].ID), nil)
	if p.status != http.StatusForbidden {
		t.Errorf("a user deleted a pet: %d", p.status)
	}
	_, body = a.signedIn(t, "admin").post("/pet/delete/"+itoa(pets[0].ID), nil)
	contains(t, body, "Pet Karabaş was deleted", "No pets found.")
}

func TestCrossSiteFormPostsAreRefused(t *testing.T) {
	a := newApp(t)
	o := store.Owner{Name: "Ali"}
	a.store.SaveOwner(context.Background(), &o) //nolint:errcheck
	admin := a.signedIn(t, "admin")

	// A form on another site that the admin's browser submits with the session cookie.
	for _, header := range [][2]string{{"Sec-Fetch-Site", "cross-site"}, {"Origin", "https://evil.example"}} {
		req, _ := http.NewRequest("POST", a.srv.URL+"/owner/delete/"+itoa(o.ID), nil)
		req.Header.Set(header[0], header[1])
		if p, _ := admin.do(req); p.status != http.StatusForbidden {
			t.Errorf("%s: status %d", header[0], p.status)
		}
	}
	if _, err := a.store.Owner(context.Background(), o.ID); err != nil {
		t.Error("the owner must still be there")
	}
}

func TestNotFound(t *testing.T) {
	a := newApp(t)
	b := a.signedIn(t, "user")
	for path, want := range map[string]string{
		"/owner/update/42":  "Owner 42 not found",
		"/owner/update/abc": "Owner abc not found",
		"/pet/update/7":     "Pet 7 not found",
		"/nothing/here":     "There is no page at /nothing/here.",
	} {
		p, body := b.get(path)
		if p.status != http.StatusNotFound {
			t.Errorf("%s: status %d", path, p.status)
		}
		contains(t, body, want)
	}
}

func TestOutputIsEscaped(t *testing.T) {
	a := newApp(t)
	b := a.signedIn(t, "user")
	_, body := b.post("/owner/create", url.Values{"name": {"<script>alert(1)</script>"}})
	if strings.Contains(body, "<script>alert(1)") {
		t.Error("names must be escaped")
	}
	contains(t, body, "&lt;script&gt;alert(1)&lt;/script&gt;")
}

func TestStaticFilesAndHealthNeedNoLogin(t *testing.T) {
	a := newApp(t)
	b := a.browser(t)
	for _, path := range []string{"/static/bootstrap/bootstrap.min.css", "/static/bootstrap/bootstrap.bundle.min.js", "/static/css/main.css", "/healthz"} {
		if p, _ := b.get(path); p.status != http.StatusOK || p.path != path {
			t.Errorf("%s: %+v", path, p)
		}
	}
	resp, err := http.Get(a.srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.Header.Get("X-Frame-Options") != "DENY" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("headers = %v", resp.Header)
	}
}

func itoa(id int64) string { return strconv.FormatInt(id, 10) }
