package kdb_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/andrebq/postigo/internal/kdb"
)

func TestSimplePutOperation(t *testing.T) {
	key := "test/key"
	firstVal := []byte("hello")
	secondVal := []byte("world")
	db, col := getCollection(t, "simple-put")
	defer db.Close()
	if err := col.Overwrite(t.Context(), key, firstVal); err != nil {
		t.Fatal(err)
	}
	if val, gen, err := col.Lookup(t.Context(), key); err != nil {
		t.Fatal(err)
	} else if gen != 1 {
		t.Fatalf("Expecting generation to be %v got %v", 1, gen)
	} else if !bytes.Equal(firstVal, val) {
		t.Fatalf("Value mismatch, expecting %v got %v", string(firstVal), string(val))
	}

	if err := col.Overwrite(t.Context(), key, secondVal); err != nil {
		t.Fatal(err)
	}
	if val, gen, err := col.Lookup(t.Context(), key); err != nil {
		t.Fatal(err)
	} else if gen != 2 {
		t.Fatalf("Expecting generation to be %v got %v", 2, gen)
	} else if !bytes.Equal(secondVal, val) {
		t.Fatalf("Value mismatch (second gen), expecting %v got %v", string(secondVal), string(val))
	}
}

func getCollection(t testing.TB, colName string) (*kdb.DB, *kdb.Collection) {
	db, err := kdb.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	col, err := db.Collection(t.Context(), colName)
	if err != nil {
		t.Fatal(err)
	}
	return db, col
}
