package objectstore

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"
)

type FakeStore struct {
	mu      sync.RWMutex
	objects map[string]fakeObject
	server  *httptest.Server
}

type fakeObject struct {
	body        []byte
	etag        string
	contentType string
}

func NewFake() *FakeStore {
	store := &FakeStore{
		objects: make(map[string]fakeObject),
	}
	store.server = httptest.NewServer(http.HandlerFunc(store.handle))
	return store
}

func (s *FakeStore) Close() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *FakeStore) PresignUpload(_ context.Context, objectKey, _ string, expires time.Duration) (*PresignedUpload, error) {
	return &PresignedUpload{
		URL:       s.server.URL + "/upload/" + url.PathEscape(objectKey),
		Method:    http.MethodPut,
		Headers:   map[string]string{},
		ExpiresAt: time.Now().UTC().Add(expires),
	}, nil
}

func (s *FakeStore) PresignDownload(_ context.Context, objectKey string, expires time.Duration) (*PresignedDownload, error) {
	return &PresignedDownload{
		URL:       s.server.URL + "/download/" + url.PathEscape(objectKey),
		ExpiresAt: time.Now().UTC().Add(expires),
	}, nil
}

func (s *FakeStore) StatObject(_ context.Context, objectKey string) (*ObjectInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	object, ok := s.objects[objectKey]
	if !ok {
		return nil, ErrObjectNotFound
	}
	return &ObjectInfo{
		ETag:        object.etag,
		Size:        int64(len(object.body)),
		ContentType: object.contentType,
	}, nil
}

func (s *FakeStore) DeleteObject(_ context.Context, objectKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, objectKey)
	return nil
}

func (s *FakeStore) handle(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/upload/"):
		s.handleUpload(w, r)
	case strings.HasPrefix(r.URL.Path, "/download/"):
		s.handleDownload(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *FakeStore) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	objectKey, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/upload/"))
	if err != nil || objectKey == "" {
		http.Error(w, "invalid object key", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	sum := md5.Sum(body)
	etag := `"` + hex.EncodeToString(sum[:]) + `"`

	s.mu.Lock()
	s.objects[objectKey] = fakeObject{
		body:        body,
		etag:        etag,
		contentType: r.Header.Get("Content-Type"),
	}
	s.mu.Unlock()

	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
}

func (s *FakeStore) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	objectKey, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/download/"))
	if err != nil || objectKey == "" {
		http.Error(w, "invalid object key", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	object, ok := s.objects[objectKey]
	s.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	if object.contentType != "" {
		w.Header().Set("Content-Type", object.contentType)
	}
	w.Header().Set("ETag", object.etag)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(object.body)))
	_, _ = w.Write(object.body)
}
