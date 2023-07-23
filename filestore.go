package main

import "io"

type StorageBackend interface {
	Exists(filename string) (bool, error)
	Read(filename string) (io.ReadCloser, error)
	Write(filename string, r io.Reader) (int64, error)
}
