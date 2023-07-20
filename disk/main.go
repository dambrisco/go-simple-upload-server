package disk

import (
	"errors"
	"io"
	"os"
	"path"
)

type Store struct {
	rootDirectory string
}

var (
	errNilStore = errors.New("method called on nil Store")
)

func NewStore(rootDirectory string) (*Store, error) {
	if rootDirectory == "" {
		return nil, errors.New("no root directory provided")
	}
	return &Store{
		rootDirectory,
	}, nil
}

func (ds *Store) Exists(filename string) (bool, error) {
	if ds == nil {
		return false, errNilStore
	}

	fPath := path.Join(ds.rootDirectory, filename)
	f, err := os.Open(fPath)
	if err != nil {
		return false, err
	}
	if f != nil {
		defer f.Close()
		return true, nil
	}
	return false, nil
}

func (ds *Store) Read(filename string) (io.ReadCloser, error) {
	if ds == nil {
		return nil, errNilStore
	}

	fPath := path.Join(ds.rootDirectory, filename)
	f, err := os.Open(fPath)
	if err != nil {
		return nil, err
	}
	if f != nil {
		return f, nil
	}
	return nil, nil
}

func (ds *Store) Write(filename string, r io.Reader) (int64, error) {
	if ds == nil {
		return 0, errNilStore
	}

	dstPath := path.Join(ds.rootDirectory, filename)
	dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return 0, err
	}
	if dstFile != nil {
		defer dstFile.Close()
		return io.Copy(dstFile, r)
	}
	return 0, nil
}
