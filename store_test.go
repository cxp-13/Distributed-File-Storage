package main

import (
	"bytes"
	"fmt"
	"testing"
)

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFun,
	}
	s := NewStore(opts)
	return s
}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}

func TestPathTransformFun(t *testing.T) {
	key := "mondasdad"
	pathName := CASPathTransformFun(key)
	fmt.Println(pathName)

}

func TestStore(t *testing.T) {
	s := newStore()
	defer teardown(t, s)
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("test_%d", i)
		data := bytes.NewReader([]byte("some peg data"))
		if err := s.writeStream(key, data); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); !ok {
			t.Error("file not found")
		}

		//bytes, err := s.Read("test_file")
		//if err != nil {
		//	t.Error(err)
		//}
		//
		//fmt.Println(string(bytes))
	}

}
