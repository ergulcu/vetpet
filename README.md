# VetPet

A small veterinary clinic application: owners and their pets, with search, create, edit and delete screens
(Bootstrap 5) behind a login. Written in Go, with the same pages as the earlier Spring Boot and Thymeleaf
version (kept at the [`java` tag](https://github.com/ergulcu/vetpet/tree/java)), as one binary without a
Java runtime.

## Run

```bash
go run ./cmd/vetpet              # http://localhost:8080, data in memory
docker compose up --build        # data in a volume
```

| User | Password | |
|---|---|---|
| `admin` | `admin` | can delete |
| `user` | `user` | can read, create and edit |

These are demo passwords; the log warns while they are in use.

| Variable | |
|---|---|
| `VETPET_ADDR` | listen address, `:8080` |
| `VETPET_DATABASE` | SQLite file, e.g. `./vetpet.db`; without it the data is kept in memory and is gone after a restart (as with H2 in the Java version) |
| `VETPET_SECURITY_ADMIN_PASSWORD`, `VETPET_SECURITY_USER_PASSWORD` | the passwords of the two accounts (same names as in the Java version) |
| `VETPET_SECURE_COOKIE` | `true` when the application is served over HTTPS |
| `VETPET_SESSION_TIMEOUT` | idle time after which a user is signed out, `30m` |

SQLite is the pure Go [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), so no C compiler is needed.

## Security

- Form login with a server side session; the cookie is `HttpOnly` and `SameSite=Lax`, and a new session id
  is issued at every sign-in. Passwords are kept as bcrypt hashes.
- Every page except the login page, the static files and `/healthz` needs a signed-in user. Deleting is a
  POST and only for admins; others get the access denied page (403).
- Form posts that another site makes the browser send (CSRF) are refused with Go's
  [`CrossOriginProtection`](https://pkg.go.dev/net/http#CrossOriginProtection), which checks the
  `Sec-Fetch-Site` and `Origin` headers that browsers send. Pages have no CSRF token field.

## Differences from the Java version

- An owner who still has pets is kept with a message, as before; a foreign key now refuses it in the database as well.
- The search ignores case for every letter (`şük` finds `Şükrü`).
- The data can be kept in a file.
- Bootstrap is embedded in the binary instead of coming from a WebJar.

## Tests

```bash
go test ./...
```

The web tests drive the application through HTTP with a cookie jar, as a browser does: login, the forms and
their errors, the admin-only delete, cross-site posts and escaping.

## License

Copyright 2019 Mehmet ERGULCU. Licensed under the [Apache License 2.0](LICENSE). Bootstrap is
Copyright 2011-2026 The Bootstrap Authors, under the [MIT License](internal/web/static/bootstrap/LICENSE).
