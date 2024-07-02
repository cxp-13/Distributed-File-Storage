package main

import (
	"bytes"
	"testing"
)

func TestPathTransformFun(t *testing.T) {

}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFun,
	}
	s := NewStore(opts)

	data := bytes.NewReader([]byte("hello world"))
	if err := s.writeStream("hello", data); err != nil {

	}

}
