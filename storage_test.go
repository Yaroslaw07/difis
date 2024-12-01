package main

import (
	"bytes"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "bestpic"
	pathKey := CASPathTransformFunc(key)

	expectedOriginalKey := "6f90c0cbffd1b2aa1e69c839a5b9606ff145c565"
	expectedPathname := "6f90c/0cbff/d1b2a/a1e69/c839a/5b960/6ff14/5c565"

	if pathKey.PathName != expectedPathname {
		t.Errorf("CASPathTransformFunc failed have: %s, want: %s", pathKey.PathName, expectedPathname)
	}

	if pathKey.Filename != expectedOriginalKey {
		t.Errorf("CASPathTransformFunc failed have: %s, want: %s", pathKey.Filename, expectedOriginalKey)
	}
}

func TestStoreDeleteKey(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)
	key := "bestpic"
	data := []byte("some data")

	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Errorf("writeStream failed: %v", err)
	}

	if err := s.Delete(key); err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)
	key := "bestpic"
	data := []byte("some data")

	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Errorf("writeStream failed: %v", err)
	}

	if ok := s.Has(key); !ok {
		t.Errorf("Has failed: %v", ok)
	}

	r, err := s.Read(key)
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}

	b, _ := io.ReadAll(r)
	if string(b) != string(data) {
		t.Errorf("want %s, have %s", data, b)
	}

	s.Delete(key)
}
