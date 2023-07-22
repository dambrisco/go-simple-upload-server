package main

import (
	"encoding/json"
	"net/http"
)

type uploadedResponse struct {
	Filename string `json:"filename"`
}

type errorResponse struct {
	Message string `json:"error"`
}

func writeError(w http.ResponseWriter, err error) (int, error) {
	return writeErrorWithMessage(w, err, err.Error())
}

func writeErrorWithMessage(w http.ResponseWriter, err error, msg string) (int, error) {
	logger.WithError(err).Error(msg)
	body := errorResponse{Message: msg}
	b, e := json.Marshal(body)
	// if an error is occured on marshaling, write empty value as response.
	if e != nil {
		return w.Write([]byte{})
	}
	return w.Write(b)
}

func writeSuccess(w http.ResponseWriter, filename string) (int, error) {
	body := uploadedResponse{Filename: filename}
	b, e := json.Marshal(body)
	// if an error is occured on marshaling, write empty value as response.
	if e != nil {
		return w.Write([]byte{})
	}
	return w.Write(b)
}
