package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "bestpic"
	pathname := CASPathTransformFunc(key)

	fmt.Println(pathname)
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: DefaultPathTransformFunc,
	}

	s := NewStore(opts)

	data := bytes.NewReader(([]byte)("some data"))

	if err := s.writeStream("key", data); err != nil {
		t.Errorf("writeStream failed: %v", err)
	}
}
