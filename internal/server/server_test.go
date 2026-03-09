package server

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"homestack/internal/auth"
	"homestack/internal/db"
	"homestack/internal/storage"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	root := t.TempDir()
	guard, err := storage.NewGuard(root)
	if err != nil {
		t.Fatalf("NewGuard: %v", err)
	}
	return &Server{guard: guard}, root
}

func withUser(r *http.Request) *http.Request {
	u := &auth.User{ID: 1, Username: "test", IsAdmin: false}
	ctx := auth.WithUser(context.Background(), u)
	return r.WithContext(ctx)
}

func TestHandleListFiles_requiresAuth(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	w := httptest.NewRecorder()
	s.requireAuth(s.handleListFiles)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleListFiles_listsEntries(t *testing.T) {
	s, root := newTestServer(t)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/files?path=", nil)
	req = withUser(req)
	w := httptest.NewRecorder()
	s.handleListFiles(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp struct {
		Entries []fileEntry `json:"entries"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(resp.Entries))
	}
}

func TestHandleUploadFiles_createsFiles(t *testing.T) {
	s, _ := newTestServer(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("path", "")
	fw, err := writer.CreateFormFile("file", "upload.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write([]byte("content")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/files/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = withUser(req)
	w := httptest.NewRecorder()
	s.handleUploadFiles(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleDownloadFile_notFound(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/files/download?path=missing.txt", nil)
	req = withUser(req)
	w := httptest.NewRecorder()
	s.handleDownloadFile(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDeletePath_deletesFile(t *testing.T) {
	s, root := newTestServer(t)
	target := filepath.Join(root, "deleteme.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	form := bytes.NewBufferString("path=deleteme.txt")
	req := httptest.NewRequest(http.MethodPost, "/api/files/delete", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = withUser(req)
	w := httptest.NewRecorder()
	s.handleDeletePath(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("file still exists or other error: %v", err)
	}
}
func TestNew_ConfiguresServerWithDependencies(t *testing.T) {
	dir := t.TempDir()
	database, err := db.Open(dir)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer database.Close()

	guard, err := storage.NewGuard(t.TempDir())
	if err != nil {
		t.Fatalf("storage.NewGuard: %v", err)
	}

	s := New(":0", database, guard)
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.db != database {
		t.Errorf("Server db mismatch")
	}
	if s.guard != guard {
		t.Errorf("Server guard mismatch")
	}
	if s.handler == nil || s.mux == nil {
		t.Errorf("Server handler or mux not initialized")
	}
}
