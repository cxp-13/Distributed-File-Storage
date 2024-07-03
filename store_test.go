package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPathTransformFun(t *testing.T) {
	key := "mondasdad"
	pathName := CASPathTransformFun(key)
	fmt.Println(pathName)

}

func TestStoreDelete(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFun,
	}
	s := NewStore(opts)
	s.Delete("cxp")
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFun,
	}
	s := NewStore(opts)

	data := bytes.NewReader([]byte("some peg data"))
	if err := s.writeStream("test_file", data); err != nil {
		t.Error(err)
	}

	if ok := s.Has("test_file"); !ok {
		t.Error("file not found")
	}

	bytes, err := s.Read("test_file")
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(bytes))
}
