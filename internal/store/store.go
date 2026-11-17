// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

// Package store keeps owners and their pets in SQLite: in memory by default, as the H2 database of the
// Java version did, or in a file so the data survives a restart.
package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync"

	"modernc.org/sqlite"
)

// Owner of pets.
type Owner struct {
	ID    int64
	Name  string
	Email string
	Phone string
}

// Pet belongs to one owner. Age may be unknown.
type Pet struct {
	ID          int64
	OwnerID     int64
	OwnerName   string // filled when read
	Name        string
	Type        string
	Kind        string
	Age         *int
	Description string
}

// ErrNotFound is returned for an unknown id.
var ErrNotFound = errors.New("not found")

// HasPetsError is returned when an owner who still has pets is deleted.
type HasPetsError struct{ Pets int }

func (e *HasPetsError) Error() string { return fmt.Sprintf("the owner still has %d pet(s)", e.Pets) }

// Store is safe for concurrent use.
type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS owner (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    name  TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS pet (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id    INTEGER NOT NULL REFERENCES owner (id) ON DELETE RESTRICT,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    kind        TEXT NOT NULL,
    age         INTEGER CHECK (age BETWEEN 0 AND 100),
    description TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS pet_owner ON pet (owner_id);
`

var registerOnce sync.Once

// Open opens the database file at path, or a database in memory when path is empty, and creates the tables.
func Open(ctx context.Context, path string) (*Store, error) {
	registerOnce.Do(func() {
		// SQLite's lower() only knows ASCII; searching "şük" must find "Şükrü".
		sqlite.MustRegisterDeterministicScalarFunction("unicode_lower", 1,
			func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
				s, _ := args[0].(string)
				return strings.ToLower(s), nil
			})
	})
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	if path == "" {
		dsn = "file::memory:?_pragma=foreign_keys(1)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if path == "" {
		// Every connection to :memory: is a database of its own, so keep exactly one.
		db.SetMaxOpenConns(1)
		db.SetConnMaxIdleTime(0)
		db.SetConnMaxLifetime(0)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// Owners returns the owners ordered by name; a non-empty search keeps those whose name contains it, ignoring case.
func (s *Store) Owners(ctx context.Context, search string) ([]Owner, error) {
	query := "SELECT id, name, email, phone FROM owner"
	var args []any
	if search = strings.TrimSpace(search); search != "" {
		query += " WHERE instr(unicode_lower(name), ?) > 0"
		args = append(args, strings.ToLower(search))
	}
	rows, err := s.db.QueryContext(ctx, query+" ORDER BY unicode_lower(name), id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var owners []Owner
	for rows.Next() {
		var o Owner
		if err := rows.Scan(&o.ID, &o.Name, &o.Email, &o.Phone); err != nil {
			return nil, err
		}
		owners = append(owners, o)
	}
	return owners, rows.Err()
}

func (s *Store) Owner(ctx context.Context, id int64) (Owner, error) {
	var o Owner
	err := s.db.QueryRowContext(ctx, "SELECT id, name, email, phone FROM owner WHERE id = ?", id).
		Scan(&o.ID, &o.Name, &o.Email, &o.Phone)
	if errors.Is(err, sql.ErrNoRows) {
		return o, ErrNotFound
	}
	return o, err
}

// SaveOwner inserts an owner with ID 0 and sets the ID, or updates an existing one.
func (s *Store) SaveOwner(ctx context.Context, o *Owner) error {
	if o.ID == 0 {
		res, err := s.db.ExecContext(ctx, "INSERT INTO owner (name, email, phone) VALUES (?, ?, ?)", o.Name, o.Email, o.Phone)
		if err != nil {
			return err
		}
		o.ID, err = res.LastInsertId()
		return err
	}
	return affected(s.db.ExecContext(ctx, "UPDATE owner SET name = ?, email = ?, phone = ? WHERE id = ?",
		o.Name, o.Email, o.Phone, o.ID))
}

// DeleteOwner refuses an owner who still has pets with a *HasPetsError.
func (s *Store) DeleteOwner(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	var pets int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM pet WHERE owner_id = ?", id).Scan(&pets); err != nil {
		return err
	}
	if pets > 0 {
		return &HasPetsError{Pets: pets}
	}
	if err := affected(tx.ExecContext(ctx, "DELETE FROM owner WHERE id = ?", id)); err != nil {
		return err
	}
	return tx.Commit()
}

const petColumns = "p.id, p.owner_id, o.name, p.name, p.type, p.kind, p.age, p.description"

// Pets returns the pets with their owner's name, ordered by name; a non-empty search keeps those whose name
// contains it, ignoring case.
func (s *Store) Pets(ctx context.Context, search string) ([]Pet, error) {
	query := "SELECT " + petColumns + " FROM pet p JOIN owner o ON o.id = p.owner_id"
	var args []any
	if search = strings.TrimSpace(search); search != "" {
		query += " WHERE instr(unicode_lower(p.name), ?) > 0"
		args = append(args, strings.ToLower(search))
	}
	rows, err := s.db.QueryContext(ctx, query+" ORDER BY unicode_lower(p.name), p.id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pets []Pet
	for rows.Next() {
		p, err := scanPet(rows)
		if err != nil {
			return nil, err
		}
		pets = append(pets, p)
	}
	return pets, rows.Err()
}

func (s *Store) Pet(ctx context.Context, id int64) (Pet, error) {
	p, err := scanPet(s.db.QueryRowContext(ctx,
		"SELECT "+petColumns+" FROM pet p JOIN owner o ON o.id = p.owner_id WHERE p.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

// SavePet inserts a pet with ID 0 and sets the ID, or updates an existing one. An unknown owner is ErrNotFound.
func (s *Store) SavePet(ctx context.Context, p *Pet) error {
	if _, err := s.Owner(ctx, p.OwnerID); err != nil {
		return err
	}
	if p.ID == 0 {
		res, err := s.db.ExecContext(ctx, `INSERT INTO pet (owner_id, name, type, kind, age, description)
			VALUES (?, ?, ?, ?, ?, ?)`, p.OwnerID, p.Name, p.Type, p.Kind, p.Age, p.Description)
		if err != nil {
			return err
		}
		p.ID, err = res.LastInsertId()
		return err
	}
	return affected(s.db.ExecContext(ctx, `UPDATE pet SET owner_id = ?, name = ?, type = ?, kind = ?, age = ?,
		description = ? WHERE id = ?`, p.OwnerID, p.Name, p.Type, p.Kind, p.Age, p.Description, p.ID))
}

func (s *Store) DeletePet(ctx context.Context, id int64) error {
	return affected(s.db.ExecContext(ctx, "DELETE FROM pet WHERE id = ?", id))
}

type scanner interface{ Scan(dest ...any) error }

func scanPet(row scanner) (Pet, error) {
	var p Pet
	var age sql.NullInt64
	err := row.Scan(&p.ID, &p.OwnerID, &p.OwnerName, &p.Name, &p.Type, &p.Kind, &age, &p.Description)
	if age.Valid {
		a := int(age.Int64)
		p.Age = &a
	}
	return p, err
}

func affected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}
