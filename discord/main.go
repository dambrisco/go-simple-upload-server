package discord

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type Store struct {
	webhookURL string
	httpClient *http.Client
}

var (
	errNilStore       = errors.New("method called on nil Store")
	errNotImplemented = errors.New("method not implemented")
)

func NewStore(webhookURL string) (*Store, error) {
	if webhookURL == "" {
		return nil, errors.New("no webhook provided")
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return &Store{
		webhookURL: webhookURL,
		httpClient: client,
	}, nil
}

func (ds *Store) Exists(filename string) (bool, error) {
	return false, errNotImplemented
}

func (ds *Store) Read(filename string) (io.ReadCloser, error) {
	return nil, errNotImplemented
}

func (ds *Store) Write(filename string, r io.Reader) (int64, error) {
	if ds == nil {
		return 0, errNilStore
	}
	b := &bytes.Buffer{}
	w := multipart.NewWriter(b)
	w.WriteField("payload_json", "{}")
	fw, err := w.CreateFormFile("file0", fmt.Sprintf("backup_%s", time.Now().Format(time.RFC3339)))
	if err != nil {
		return 0, err
	}
	filesize, err := io.Copy(fw, r)
	if err != nil {
		return 0, err
	}
	w.Close()
	resp, err := ds.httpClient.Post(ds.webhookURL, w.FormDataContentType(), b)
	if err != nil {
		return 0, err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return 0, fmt.Errorf("non-2xx response status code received: %d", resp.StatusCode)
	}
	return filesize, nil
}
