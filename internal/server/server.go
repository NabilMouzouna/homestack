package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"homestack/internal/auth"
	"homestack/internal/db"
	"homestack/internal/storage"
	"homestack/web"
)

// Server holds HTTP handlers and dependencies.
type Server struct {
	Addr      string
	db        *db.DB
	guard     *storage.Guard
	store     *auth.Store
	templates *template.Template
	mux       *http.ServeMux
	handler   http.Handler
}

// New creates a server that uses the given DB and storage guard.
func New(addr string, database *db.DB, guard *storage.Guard) *Server {
	tpls := template.Must(template.ParseFS(web.Templates(), "templates/*.html"))

	s := &Server{
		Addr:      addr,
		db:        database,
		guard:     guard,
		store:     auth.NewStore(database),
		templates: tpls,
	}

	mux := http.NewServeMux()

	// Static assets (HTMX, CSS, etc.)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.Static()))))

	// Public login page + form submit.
	mux.HandleFunc("GET /login", s.handleLoginGet)
	mux.HandleFunc("POST /login", s.handleLoginPost)

	// Root redirects to login (or files later when we have session-aware routing).
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	// Dashboard and admin (UI only for now; data wiring comes in later tasks).
	mux.HandleFunc("GET /files", s.handleDashboard)
	mux.HandleFunc("GET /admin", s.handleAdmin)

	// File APIs (JSON) — all require authenticated user.
	mux.HandleFunc("GET /api/files", s.requireAuth(s.handleListFiles))
	mux.HandleFunc("POST /api/files/upload", s.requireAuth(s.handleUploadFiles))
	mux.HandleFunc("GET /api/files/download", s.requireAuth(s.handleDownloadFile))
	mux.HandleFunc("POST /api/files/delete", s.requireAuth(s.handleDeletePath))
	mux.HandleFunc("POST /api/files/mkdir", s.requireAuth(s.handleMkdir))

	s.mux = mux
	s.handler = sessionMiddleware(s.store)(mux)
	return s
}

// sessionMiddleware sets the current user (and IsAdmin) on the request context from the session cookie.
func sessionMiddleware(store *auth.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if c, err := r.Cookie(auth.CookieName); err == nil && c != nil {
				token = c.Value
			}
			user, _ := store.LookupSession(token)
			ctx := auth.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// requireAuth wraps a handler and ensures a logged-in user is present.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if auth.UserFromContext(r.Context()) == nil {
			s.writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	}
}

// errorResponse matches the JSON error shape for APIs.
type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string) {
	var resp errorResponse
	resp.Error.Code = code
	resp.Error.Message = message
	s.writeJSON(w, status, resp)
}

type loginPageData struct {
	Error string
}

type dashboardPageData struct {
	User *auth.User
}

func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate(w, r, "login.html", loginPageData{})
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.renderTemplate(w, r, "login.html", loginPageData{Error: "Something went wrong. Please try again."})
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		s.renderTemplate(w, r, "login.html", loginPageData{Error: "Username and password are required."})
		return
	}
	u, err := s.db.GetUserByUsername(username)
	if err != nil || auth.ComparePassword(u.PasswordHash, password) != nil {
		w.WriteHeader(http.StatusUnauthorized)
		s.renderTemplate(w, r, "login.html", loginPageData{Error: "Invalid username or password."})
		return
	}
	token, expiresAt, err := s.store.CreateSession(u.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.renderTemplate(w, r, "login.html", loginPageData{Error: "Something went wrong. Please try again."})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // set true when serving over HTTPS
	})
	http.Redirect(w, r, "/files", http.StatusSeeOther)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := dashboardPageData{
		User: auth.UserFromContext(r.Context()),
	}
	s.renderTemplate(w, r, "dashboard.html", data)
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if !auth.IsAdmin(r.Context()) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	s.renderTemplate(w, r, "admin.html", nil)
}

// fileEntry is the JSON DTO returned by list API.
type fileEntry struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`                  // "file" or "folder"
	Size        int64     `json:"size,omitempty"`        // bytes (files only)
	ModifiedAt  time.Time `json:"modified_at,omitempty"` // best-effort
	RelPath     string    `json:"-"`                     // relative path from storage root
	DisplaySize string    `json:"display_size,omitempty"`
}

type fileListView struct {
	Path    string
	Entries []fileEntry
}

func formatSize(size int64) string {
	const (
		kb = 1024
		mb = 1024 * 1024
	)

	switch {
	case size >= mb:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(mb))
	case size >= kb:
		return fmt.Sprintf("%.1f kB", float64(size)/float64(kb))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func isHXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// buildFileList builds a file list view for a relative path under the storage root.
// On error it returns a non-zero HTTP status and error code/message.
func (s *Server) buildFileList(relPath string) (fileListView, int, string, string) {
	view := fileListView{Path: relPath}

	absPath, err := s.guard.Validate(relPath)
	if err != nil {
		return view, http.StatusBadRequest, "invalid_path", "path is invalid or escapes storage root"
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return view, http.StatusNotFound, "not_found", "path not found"
		}
		return view, http.StatusInternalServerError, "internal_error", "could not stat path"
	}
	if !info.IsDir() {
		return view, http.StatusBadRequest, "not_directory", "path is not a directory"
	}
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return view, http.StatusInternalServerError, "internal_error", "could not read directory"
	}
	for _, e := range entries {
		fe := fileEntry{Name: e.Name()}
		fi, err := e.Info()
		if err == nil {
			if fi.IsDir() {
				fe.Type = "folder"
			} else {
				fe.Type = "file"
				fe.Size = fi.Size()
				if fe.Size > 0 {
					fe.DisplaySize = formatSize(fe.Size)
				}
			}
			fe.ModifiedAt = fi.ModTime()
		} else {
			if e.IsDir() {
				fe.Type = "folder"
			} else {
				fe.Type = "file"
			}
		}
		if relPath == "" {
			fe.RelPath = fe.Name
		} else {
			fe.RelPath = filepath.Join(relPath, fe.Name)
		}
		view.Entries = append(view.Entries, fe)
	}
	return view, 0, "", ""
}

// renderFileListHTML renders the file list fragment for HTMX requests.
func (s *Server) renderFileListHTML(w http.ResponseWriter, view fileListView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "files_list.html", view); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// handleListFiles lists directory contents under the storage root.
func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	relPath := r.URL.Query().Get("path")
	view, status, code, msg := s.buildFileList(relPath)
	if status != 0 {
		s.writeError(w, status, code, msg)
		return
	}
	if isHXRequest(r) {
		s.renderFileListHTML(w, view)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"entries": view.Entries})
}

// handleUploadFiles accepts multipart file uploads under a validated directory.
func (s *Server) handleUploadFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MB default limit
		s.writeError(w, http.StatusBadRequest, "bad_request", "invalid multipart form data")
		return
	}
	relDir := r.FormValue("path")
	dirPath, err := s.guard.Validate(relDir)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_path", "path is invalid or escapes storage root")
		return
	}
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		s.writeError(w, http.StatusInternalServerError, "internal_error", "could not ensure target directory")
		return
	}
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		s.writeError(w, http.StatusBadRequest, "bad_request", "no files uploaded")
		return
	}
	var saved []string
	for _, fh := range files {
		name := filepath.Base(fh.Filename)
		if name == "" || name == "." || name == string(filepath.Separator) {
			continue
		}
		src, err := fh.Open()
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "internal_error", "could not open uploaded file")
			return
		}
		defer src.Close()
		dstPath := filepath.Join(dirPath, name)
		dst, err := os.Create(dstPath)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "internal_error", "could not create file on disk")
			return
		}
		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			s.writeError(w, http.StatusInternalServerError, "internal_error", "could not write file to disk")
			return
		}
		_ = dst.Close()
		saved = append(saved, name)
	}
	if len(saved) == 0 {
		s.writeError(w, http.StatusBadRequest, "bad_request", "no valid files uploaded")
		return
	}
	if isHXRequest(r) {
		view, status, code, msg := s.buildFileList(relDir)
		if status != 0 {
			s.writeError(w, status, code, msg)
			return
		}
		s.renderFileListHTML(w, view)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"files": saved,
	})
}

// handleDownloadFile streams a validated file from the storage root.
func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	relPath := r.URL.Query().Get("path")
	absPath, err := s.guard.Validate(relPath)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_path", "path is invalid or escapes storage root")
		return
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.writeError(w, http.StatusNotFound, "not_found", "file not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "internal_error", "could not stat file")
		return
	}
	if info.IsDir() {
		s.writeError(w, http.StatusBadRequest, "not_file", "path is a directory, not a file")
		return
	}
	http.ServeFile(w, r, absPath)
}

// handleDeletePath deletes a validated file or (empty) directory under the storage root.
func (s *Server) handleDeletePath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "invalid form data")
		return
	}
	relPath := r.FormValue("path")
	if relPath == "" {
		s.writeError(w, http.StatusBadRequest, "bad_request", "path is required")
		return
	}
	absPath, err := s.guard.Validate(relPath)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_path", "path is invalid or escapes storage root")
		return
	}
	if err := os.Remove(absPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.writeError(w, http.StatusNotFound, "not_found", "path not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "internal_error", "could not delete path")
		return
	}
	if isHXRequest(r) {
		dir := filepath.Dir(relPath)
		if dir == "." {
			dir = ""
		}
		view, status, code, msg := s.buildFileList(dir)
		if status != 0 {
			s.writeError(w, status, code, msg)
			return
		}
		s.renderFileListHTML(w, view)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleMkdir creates a new directory under the storage root.
func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "invalid form data")
		return
	}
	parentRel := r.FormValue("path")
	name := r.FormValue("name")
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "bad_request", "folder name is required")
		return
	}

	// Ensure the parent directory is within the guarded storage root.
	parentAbs, err := s.guard.Validate(parentRel)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_path", "path is invalid or escapes storage root")
		return
	}

	// Sanitize the folder name to avoid path traversal.
	safeName := filepath.Base(name)
	if safeName == "" || safeName == "." || safeName == ".." {
		s.writeError(w, http.StatusBadRequest, "bad_request", "invalid folder name")
		return
	}

	target := filepath.Join(parentAbs, safeName)
	if err := os.MkdirAll(target, 0o755); err != nil {
		s.writeError(w, http.StatusInternalServerError, "internal_error", "could not create folder")
		return
	}

	if isHXRequest(r) {
		view, status, code, msg := s.buildFileList(parentRel)
		if status != 0 {
			s.writeError(w, status, code, msg)
			return
		}
		s.renderFileListHTML(w, view)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) renderTemplate(w http.ResponseWriter, r *http.Request, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.Addr, s.handler)
}
