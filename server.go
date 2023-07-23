package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/redditdota2league/go-simple-upload-server/disk"
	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
)

var (
	errTokenMismatch   = errors.New("token mismatched")
	errMissingToken    = errors.New("missing token")
	errInvalidFilename = errors.New("invalid filename")
)

// *Server represents a simple-upload server.
type Server struct {
	chi.Router

	FileStore FileStore
	// MaxUploadSize limits the size of the uploaded content, specified with "byte".
	MaxUploadSize    int64
	SecureToken      string
	EnableCORS       bool
	ProtectedMethods []string
}

// NewServer creates a new simple-upload server.
func NewServer(documentRoot string, maxUploadSize int64, token string, enableCORS bool, protectedMethods []string) (*Server, error) {
	store, err := disk.NewStore(documentRoot)
	if err != nil {
		return nil, err
	}

	s := &Server{
		FileStore:        store,
		MaxUploadSize:    maxUploadSize,
		SecureToken:      token,
		EnableCORS:       enableCORS,
		ProtectedMethods: protectedMethods,
	}

	r := chi.NewRouter()
	r.Use(closeBody)
	r.Use(s.authorize)
	r.Use(s.handleCORS)
	r.Use(s.setMaxBytes)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Route("/files", func(r chi.Router) {
		r.Options("/", s.getOptions)
		r.Head("/{filename}", s.checkIfFileExists)
		r.Get("/{filename}", s.retrieveFile)
		r.Put("/{filename}", s.uploadFile)
		r.Post("/", s.uploadFileWithoutName)
	})

	s.Router = r
	return s, nil
}

// Handlers

func (s *Server) getOptions(w http.ResponseWriter, r *http.Request) {
	allowedMethods := []string{http.MethodPost, http.MethodPut, http.MethodGet, http.MethodHead}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ","))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) checkIfFileExists(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if !isValidFilename(filename) {
		return
	}

	exists, err := s.FileStore.Exists(filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeErrorWithMessage(w, err, "failed to check if file exists")
		return
	}
	if exists {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) retrieveFile(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if !isValidFilename(filename) {
		return
	}

	rc, err := s.FileStore.Read(filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeErrorWithMessage(w, err, "failed to read file from FileStore")
		return
	}
	defer rc.Close()
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, rc)
}

func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if !isValidFilename(filename) {
		return
	}
	writeFile(filename, s.FileStore, w, r)
}

func (s *Server) uploadFileWithoutName(w http.ResponseWriter, r *http.Request) {
	id, err := ksuid.NewRandom()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeErrorWithMessage(w, err, "failed to generate KSUID")
		return
	}
	writeFile(id.String(), s.FileStore, w, r)
}

func writeFile(filename string, fileStore FileStore, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	n, err := fileStore.Write(filename, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeErrorWithMessage(w, err, "failed to write body to filestore")
		return
	}

	logger.WithFields(logrus.Fields{
		"filename": filename,
		"size":     n,
	}).Infof("file uploaded by %s", r.Method)
	w.WriteHeader(http.StatusCreated)
	writeSuccess(w, filename)
}

// Middleware

func (s *Server) authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.isAuthenticationRequired(r) {
			token, err := getToken(r)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				writeError(w, err)
				return
			}
			if err := checkToken(s.SecureToken, token); err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				writeError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		}
	})
}

func closeBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.EnableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setMaxBytes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, s.MaxUploadSize)
		next.ServeHTTP(w, r)
	})
}

// Helpers

func (s *Server) isAuthenticationRequired(r *http.Request) bool {
	for _, m := range s.ProtectedMethods {
		if m == r.Method {
			return true
		}
	}
	return false
}

func getToken(r *http.Request) (string, error) {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	before, after, found := strings.Cut(authorization, "Bearer ")
	if !found || before != "" {
		return "", errors.New("invalid authorization format, expected \"Authorization: Bearer <token>\"")
	}
	return after, nil
}

func checkToken(serverToken, userToken string) error {
	if userToken == "" {
		return errMissingToken
	}
	if userToken != serverToken {
		return errTokenMismatch
	}
	return nil
}

func isValidFilename(input string) bool {
	if strings.Contains(input, "/") {
		return false
	}
	if strings.Contains(input, "..") {
		return false
	}
	return true
}
