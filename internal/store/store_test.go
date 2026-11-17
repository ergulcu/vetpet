// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

var ctx = context.Background()

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func owner(t *testing.T, s *Store, name string) Owner {
	t.Helper()
	o := Owner{Name: name, Email: "x@example.com"}
	if err := s.SaveOwner(ctx, &o); err != nil {
		t.Fatal(err)
	}
	return o
}

func pet(t *testing.T, s *Store, o Owner, name string) Pet {
	t.Helper()
	age := 3
	p := Pet{OwnerID: o.ID, Name: name, Type: "Dog", Kind: "Kangal", Age: &age}
	if err := s.SavePet(ctx, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOwners(t *testing.T) {
	s := open(t)
	o := owner(t, s, "Zeynep")
	owner(t, s, "ahmet")
	owner(t, s, "Şükrü")

	all, err := s.Owners(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].Name != "ahmet" || all[1].Name != "Zeynep" {
		t.Errorf("owners are ordered by name ignoring case: %+v", all)
	}

	for search, want := range map[string]int{"şük": 1, "ŞÜK": 1, "e": 2, " zey ": 1, "%": 0, "nobody": 0} {
		if got, _ := s.Owners(ctx, search); len(got) != want {
			t.Errorf("search %q found %d, want %d", search, len(got), want)
		}
	}

	o.Name, o.Phone = "Zeynep Kaya", "0555"
	if err := s.SaveOwner(ctx, &o); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Owner(ctx, o.ID); got != o {
		t.Errorf("got %+v, want %+v", got, o)
	}
	if _, err := s.Owner(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown owner: %v", err)
	}
	if err := s.SaveOwner(ctx, &Owner{ID: 99, Name: "Ghost"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("update of an unknown owner: %v", err)
	}
}

func TestPets(t *testing.T) {
	s := open(t)
	ali := owner(t, s, "Ali")
	veli := owner(t, s, "Veli")
	karabas := pet(t, s, ali, "Karabaş")
	p := Pet{OwnerID: veli.ID, Name: "Boncuk", Type: "Cat", Kind: "Van"}
	if err := s.SavePet(ctx, &p); err != nil {
		t.Fatal(err)
	}

	all, err := s.Pets(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].Name != "Boncuk" || all[0].OwnerName != "Veli" || all[0].Age != nil ||
		all[1].OwnerName != "Ali" || *all[1].Age != 3 {
		t.Errorf("pets = %+v", all)
	}
	if got, _ := s.Pets(ctx, "KARA"); len(got) != 1 || got[0].ID != karabas.ID {
		t.Errorf("search: %+v", got)
	}

	karabas.OwnerID = veli.ID
	if err := s.SavePet(ctx, &karabas); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Pet(ctx, karabas.ID); got.OwnerName != "Veli" {
		t.Errorf("the pet moved to Veli: %+v", got)
	}

	if err := s.SavePet(ctx, &Pet{OwnerID: 99, Name: "Nobody's", Type: "Dog", Kind: "?"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown owner: %v", err)
	}
	if err := s.DeletePet(ctx, karabas.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Pet(ctx, karabas.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted pet: %v", err)
	}
	if err := s.DeletePet(ctx, karabas.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
}

func TestAnOwnerWithPetsIsKept(t *testing.T) {
	s := open(t)
	ali := owner(t, s, "Ali")
	p := pet(t, s, ali, "Karabaş")

	var hasPets *HasPetsError
	if err := s.DeleteOwner(ctx, ali.ID); !errors.As(err, &hasPets) || hasPets.Pets != 1 {
		t.Fatalf("err = %v", err)
	}
	// The foreign key refuses it as well, should anything bypass the check.
	if _, err := s.db.ExecContext(ctx, "DELETE FROM owner WHERE id = ?", ali.ID); err == nil {
		t.Error("the foreign key must refuse the delete")
	}

	if err := s.DeletePet(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteOwner(ctx, ali.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteOwner(ctx, ali.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
}

func TestAFileKeepsTheData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vetpet.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	owner(t, s, "Ali")
	s.Close()

	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if got, _ := s.Owners(ctx, ""); len(got) != 1 {
		t.Errorf("owners after reopening: %+v", got)
	}
}
