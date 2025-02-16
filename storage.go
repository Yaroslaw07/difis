package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRootFolderName = "difistore"

type PathKey struct {
	PathName string
	Filename string
}

func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, "/")

	if len(paths) == 0 {
		return ""
	}

	return paths[0]
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.Filename)
}

type PathTransformFunc func(string) PathKey

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		Filename: key,
	}
}

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashStr) / blockSize

	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		PathName: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

type StoreOpts struct {
	//Root is name the of the directory, where all the files of the system will be stored
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}

	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	return s.writeStream(id, key, r)
}

func (s *Store) WriteDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {
	file, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}

	numbOfBytes, err := copyEncrypt(encKey, r, file)
	return int64(numbOfBytes), err
}

func (s *Store) Read(id string, key string) (int64, io.Reader, error) {
	return s.readStream(id, key)
}

func (s *Store) Delete(id string, key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted: [%s] from disk", pathKey.Filename)
	}()

	fistPathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FirstPathName())

	return os.RemoveAll(fistPathNameWithRoot)
}

func (s *Store) Has(id string, key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) openFileForWriting(id string, key string) (*os.File, error) {
	PathKey := s.PathTransformFunc(key)
	PathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, PathKey.PathName)

	if err := os.MkdirAll(PathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, PathKey.FullPath())
	return os.Create(fullPathWithRoot)
}

func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {
	file, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}

	return io.Copy(file, r)
}

func (s *Store) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, err
}
