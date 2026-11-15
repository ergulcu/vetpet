# VetPet

A small veterinary clinic application on Spring Boot: owners and their pets, with search, create, edit and
delete screens (Thymeleaf and Bootstrap 5) behind a login.

## Run

```bash
./mvnw spring-boot:run      # Java 25; http://localhost:8080
```

| User | Password | |
|---|---|---|
| `admin` | `admin` | can delete |
| `user` | `user` | can read, create and edit |

These are demo passwords. Set others with `VETPET_SECURITY_ADMIN_PASSWORD` and
`VETPET_SECURITY_USER_PASSWORD`. The data is kept in an in-memory H2 database and is gone after a restart.

## Fixed in this version

- The settings were YAML in a file named `application.properties`, so none of them applied; the file is
  `application.yml` now.
- Deleting was a GET link that any page could trigger; it is a POST with a CSRF token now.
- The pages referred to templates that did not exist (`user/index`, `error/access-denied`).
- `#httpServletRequest` is gone from Thymeleaf 3.1; the signed-in user is put into the model instead.
- An owner who still has pets can no longer be deleted (it failed with a database error).
- Bootstrap comes as a WebJar instead of copied files.
- Spring Boot 2.1 / Java 8 to Spring Boot 4.1 / Java 25, with `SecurityFilterChain` instead of the
  removed `WebSecurityConfigurerAdapter`.

## Tests

```bash
./mvnw verify
```

## License

Copyright 2019 Mehmet ERGULCU. Licensed under the [Apache License 2.0](LICENSE). Earlier versions were
published under the GPL v3.
