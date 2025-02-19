package server

import "io"

func (fs *FileServer) SaveLocally(key string, r io.Reader) error {
	_, err := fs.store.Write(fs.ID, key, r)
	return err
}

func (fs *FileServer) LoadLocally(key string) (int64, io.Reader, error) {
	n, r, err := fs.store.Read(fs.ID, key)
	return n, r, err
}

func (fs *FileServer) DeleteLocally(key string) error {
	return fs.store.Delete(fs.ID, key)
}
