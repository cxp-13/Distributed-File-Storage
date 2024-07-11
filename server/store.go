package server

import (
	"crypto/sha1"
	"distribute-system/crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var defaultRootName = "tmp/Store"

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

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
	_, err := os.Stat(fullPathWithRoot)

	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted %s", pathKey.FullPath())
	}()

	firstPathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FirstPathName())

	return os.RemoveAll(firstPathNameWithRoot)
}

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}
	return io.Copy(f, r)
}

func (s *Store) WriteDecrypt(encKey []byte, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}
	n, err := crypto.CopyDecrypt(encKey, r, f)
	return int64(n), err
}

func (s *Store) openFileForWriting(key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.Pathname)

	log.Printf("key:%v pathNameWithRoot:%v", key, pathNameWithRoot)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPath := pathKey.FullPath()
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, fullPath)

	return os.Create(fullPathWithRoot)
}

func (s *Store) Read(key string) (*os.File, error) {
	return s.readStream(key)
}

func (s *Store) readStream(key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
	return os.Open(fullPathWithRoot)
}
