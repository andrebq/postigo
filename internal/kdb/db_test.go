package kdb_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/andrebq/postigo/internal/kdb"
)

type (
	Machine struct {
		ID   string
		FQDN string
	}
)

func (m Machine) GetID() string { return m.ID }

func TestSimplePutOperation(t *testing.T) {
	db, err := kdb.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := kdb.GetCollection[Machine](t.Context(), db, "machines")
	m := Machine{
		ID:   "node1",
		FQDN: "node-1.example.com",
	}
	if err := col.Put(t.Context(), &m); err != nil {
		t.Fatal(err)
	}
	var fromDB Machine
	err = col.Lookup(t.Context(), &fromDB, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromDB, m) {
		t.Fatalf("Expecting %v got %v", m, fromDB)
	}
}
