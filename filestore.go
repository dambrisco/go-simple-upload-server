package main

import "io"

type FileStore interface {
	Exists(filename string) (bool, error)
	Read(filename string) (io.ReadCloser, error)
	Write(filename string, r io.Reader) (int64, error)
}
