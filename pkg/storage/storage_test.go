package storage

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/Yaroslaw07/difis/pkg/crypto"
)

func TestPathTransformFunc(t *testing.T) {
	key := "bestpic"
	pathKey := CASPathTransformFunc(key)

	expectedFilename := "6f90c0cbffd1b2aa1e69c839a5b9606ff145c565"
	expectedPathname := "6f90c/0cbff/d1b2a/a1e69/c839a/5b960/6ff14/5c565"

	if pathKey.PathName != expectedPathname {
		t.Errorf("CASPathTransformFunc failed have: %s, want: %s", pathKey.PathName, expectedPathname)
	}

	if pathKey.Filename != expectedFilename {
		t.Errorf("CASPathTransformFunc failed have: %s, want: %s", pathKey.Filename, expectedFilename)
	}
}

func TestStore(t *testing.T) {
	s := newStore()
	id := crypto.GenerateID()
	defer teardown(t, s)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key-%d", i)
		data := []byte("some data")

		if _, err := s.Write(id, key, bytes.NewReader(data)); err != nil {
			t.Errorf("writeStream failed: %v", err)
		}

		if ok := s.Has(id, key); !ok {
			t.Errorf("Has failed: %v", ok)
		}

		_, r, err := s.Read(id, key)
		if err != nil {
			t.Errorf("Read failed: %v", err)
		}

		b, _ := io.ReadAll(r)
		if string(b) != string(data) {
			t.Errorf("want %s, have %s", data, b)
		}

		if err := s.Delete(id, key); err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		if ok := s.Has(id, key); ok {
			t.Errorf("Has detected deleted key  %v", key)
		}
	}
}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStore(opts)
}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Errorf("Clear failed: %v", err)
	}
}
