package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var defaultRootName = "tmp/store"

func CASPathTransformFun(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blocksize := 5
	sliceLen := len(hashStr) / blocksize

	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i+1)*blocksize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		Pathname: strings.Join(paths, "/"),
		Filename: key,
	}

}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	Pathname string
	Filename string
}

func (k PathKey) FirstPathName() string {
	split := strings.Split(k.Pathname, "/")
	if len(split) == 0 {
		return ""
	}
	return split[0]
}

func (k PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", k.Pathname, k.Filename)
}

type StoreOpts struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		Pathname: key,
		Filename: key,
	}
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootName

	}
	return &Store{
		StoreOpts: opts,
	}
}

func (s Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
	_, err := os.Stat(fullPathWithRoot)

	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

func (s Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted %s", pathKey.FullPath())
	}()

	firstPathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FirstPathName())

	return os.RemoveAll(firstPathNameWithRoot)
}

func (s Store) Read(key string) ([]byte, error) {
	stream, err := s.readStream(key)
	if err != nil {
		return nil, err
	}

	defer stream.Close()
	b := new(bytes.Buffer)
	b.ReadFrom(stream)
	return b.Bytes(), nil
}

func (s Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
	return os.Open(fullPathWithRoot)

}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.Pathname)

	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return err
	}

	fullPath := pathKey.FullPath()
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, fullPath)

	f, err := os.Create(fullPathWithRoot)

	if err != nil {
		return err
	}
	n, err := io.Copy(f, r)

	if err != nil {
		return err
	}

	log.Printf("written %d bytes to %s", n, pathKey.Filename)

	return nil

}
