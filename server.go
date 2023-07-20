package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/redditdota2league/go-simple-upload-server/disk"
	"github.com/sirupsen/logrus"
)

var (
	rePathFiles = regexp.MustCompile(`^/files(/.*)?(/[^/]+)$`)

	errTokenMismatch = errors.New("token mismatched")
	errMissingToken  = errors.New("missing token")
)

// *Server represents a simple-upload server.
type Server struct {
	FileStore FileStore
	// MaxUploadSize limits the size of the uploaded content, specified with "byte".
	MaxUploadSize    int64
	SecureToken      string
	EnableCORS       bool
	ProtectedMethods []string
}

// NewServer creates a new simple-upload server.
func NewServer(documentRoot string, maxUploadSize int64, token string, enableCORS bool, protectedMethods []string) (*Server, error) {
	s, err := disk.NewStore(documentRoot)
	if err != nil {
		return nil, err
	}
	return &Server{
		FileStore:        s,
		MaxUploadSize:    maxUploadSize,
		SecureToken:      token,
		EnableCORS:       enableCORS,
		ProtectedMethods: protectedMethods,
	}, nil
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	if !rePathFiles.MatchString(r.URL.Path) {
		w.WriteHeader(http.StatusNotFound)
		writeError(w, fmt.Errorf("\"%s\" is not found", r.URL.Path))
		return
	}
	if s.EnableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	matches := rePathFiles.FindStringSubmatch(r.URL.Path)
	if matches == nil {
		logger.WithField("path", r.URL.Path).Info("invalid path")
		w.WriteHeader(http.StatusNotFound)
		writeError(w, fmt.Errorf("\"%s\" is not found", r.URL.Path))
		return
	}
	targetPath := path.Clean(path.Join(matches[1], matches[2]))

	rc, err := s.FileStore.Read(targetPath)
	if err != nil {
		logger.WithError(err).Error("failed to read file from FileStore")
		w.WriteHeader(http.StatusInternalServerError)
		writeError(w, err)
		return
	}
	defer rc.Close()
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, rc)
}

func (s *Server) handlePut(w http.ResponseWriter, r *http.Request) {
	matches := rePathFiles.FindStringSubmatch(r.URL.Path)
	if matches == nil {
		logger.WithField("path", r.URL.Path).Info("invalid path")
		w.WriteHeader(http.StatusNotFound)
		writeError(w, fmt.Errorf("\"%s\" is not found", r.URL.Path))
		return
	}
	targetPath := path.Clean(path.Join(matches[1], matches[2]))

	reader := http.MaxBytesReader(w, r.Body, s.MaxUploadSize)
	defer reader.Close()
	n, err := s.FileStore.Write(targetPath, r.Body)
	if err != nil {
		logger.WithError(err).Error("failed to write body to the file")
		w.WriteHeader(http.StatusInternalServerError)
		writeError(w, err)
		return
	}

	logger.WithFields(logrus.Fields{
		"path": r.URL.Path,
		"size": n,
	}).Infof("file uploaded by %s", r.Method)
	if s.EnableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.WriteHeader(http.StatusCreated)
	writeSuccess(w, r.URL.Path)
}

func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {
	var allowedMethods []string
	if rePathFiles.MatchString(r.URL.Path) {
		allowedMethods = []string{http.MethodPost, http.MethodPut, http.MethodGet, http.MethodHead}
	} else {
		w.WriteHeader(http.StatusNotFound)
		writeError(w, errors.New("not found"))
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ","))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) checkToken(r *http.Request) error {
	// first, try to get the token from the query strings
	token := r.URL.Query().Get("token")
	// if token is not found, check the form parameter.
	if token == "" {
		token = r.FormValue("token")
	}
	if token == "" {
		return errMissingToken
	}
	if token != s.SecureToken {
		return errTokenMismatch
	}
	return nil
}

func (s *Server) isAuthenticationRequired(r *http.Request) bool {
	for _, m := range s.ProtectedMethods {
		if m == r.Method {
			return true
		}
	}
	return false
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if err := s.checkToken(r); s.isAuthenticationRequired(r) && err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		writeError(w, err)
		return
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		s.handleGet(w, r)
	case http.MethodPost, http.MethodPut:
		s.handlePut(w, r)
	case http.MethodOptions:
		s.handleOptions(w, r)
	default:
		w.Header().Add("Allow", "GET,HEAD,POST,PUT")
		w.WriteHeader(http.StatusMethodNotAllowed)
		writeError(w, fmt.Errorf("method \"%s\" is not allowed", r.Method))
	}
}
