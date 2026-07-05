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

func TestCASOperation(t *testing.T) {
	db, err := kdb.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := kdb.GetCollection[Machine](t.Context(), db, "machines")
	m := Machine{
		ID:   "node1",
		FQDN: "node-1.example",
	}
	if err := col.Put(t.Context(), &m); err != nil {
		t.Fatal(err)
	}

	updated, err := col.CAS(t.Context(), "node1", func(m *Machine) (*Machine, error) {
		m.FQDN = m.FQDN + ".com"
		return m, nil
	})
	if err != nil {
		t.Fatal(err)
	} else if !updated {
		t.Fatal("should have updated the record")
	}
	var fromdb Machine
	if err := col.Lookup(t.Context(), &fromdb, m.ID); err != nil {
		t.Fatal(err)
	}

	updated, err = col.CAS(t.Context(), "node1", func(m *Machine) (*Machine, error) {
		// this function runs outside of a transaction and might be called
		// multiple times.
		// executing a put operation here is the same as a different thread
		// updating the database concurrently
		return m, col.Put(t.Context(), m)
	})
	if updated {
		t.Fatal("CAS operation overwrote a value")
	} else if !kdb.IsConcurrentUpdate(err) {
		t.Fatalf("Error should be concurrent update but got %v", err)
	}
}
