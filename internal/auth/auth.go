package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Fupace/hermes-go-jumpserver/internal/model"
)

type Store struct {
	db    *sql.DB
	mu    sync.RWMutex
	cache map[string]string
}

func NewStore(dbPath string) *Store {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		panic(err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, username TEXT UNIQUE, password TEXT)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS sessions (id TEXT PRIMARY KEY, username TEXT, created_at DATETIME)`)

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if count == 0 {
		db.Exec(`INSERT INTO users (username, password) VALUES (?, ?)`, "admin", "admin")
	}

	return &Store{
		db:    db,
		cache: make(map[string]string),
	}
}

func (s *Store) ValidateSession(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	s.mu.RLock()
	_, ok := s.cache[sessionID]
	s.mu.RUnlock()
	if ok {
		return true
	}
	var username string
	err := s.db.QueryRow(`SELECT username FROM sessions WHERE id = ?`, sessionID).Scan(&username)
	if err == nil {
		s.mu.Lock()
		s.cache[sessionID] = username
		s.mu.Unlock()
		return true
	}
	return false
}

func (s *Store) Authenticate(username, password string) (string, bool) {
	var stored string
	err := s.db.QueryRow(`SELECT password FROM users WHERE username = ?`, username).Scan(&stored)
	if err != nil || stored != password {
		return "", false
	}
	sessionID := generateID()
	s.db.Exec(`INSERT INTO sessions (id, username, created_at) VALUES (?, ?, ?)`, sessionID, username, time.Now())
	s.mu.Lock()
	s.cache[sessionID] = username
	s.mu.Unlock()
	return sessionID, true
}

func (s *Store) Logout(sessionID string) {
	s.db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
	s.mu.Lock()
	delete(s.cache, sessionID)
	s.mu.Unlock()
}

func (s *Store) GetUsername(sessionID string) string {
	s.mu.RLock()
	u, ok := s.cache[sessionID]
	s.mu.RUnlock()
	if ok {
		return u
	}
	var username string
	s.db.QueryRow(`SELECT username FROM sessions WHERE id = ?`, sessionID).Scan(&username)
	return username
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type Handler struct {
	store *Store
}

func NewHandler(s *Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.LoginResponse{Success: false, Message: "invalid request"})
		return
	}
	sessionID, ok := h.store.Authenticate(req.Username, req.Password)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, model.LoginResponse{Success: false, Message: "invalid credentials"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400,
	})
	writeJSON(w, http.StatusOK, model.LoginResponse{Token: sessionID, Success: true})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		h.store.Logout(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *Handler) Session(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil || !h.store.ValidateSession(cookie.Value) {
		writeJSON(w, http.StatusUnauthorized, map[string]bool{"authenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"username":      h.store.GetUsername(cookie.Value),
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
