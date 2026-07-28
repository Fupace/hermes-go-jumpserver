package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Fupace/hermes-go-jumpserver/internal/auth"
	"github.com/Fupace/hermes-go-jumpserver/internal/handler"
	"github.com/Fupace/hermes-go-jumpserver/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dataDir := flag.String("data-dir", "/data", "Data directory for SQLite DB and SSH keys")
	flag.Parse()

	os.MkdirAll(*dataDir, 0755)

	dbPath := filepath.Join(*dataDir, "jumpserver.db")
	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer s.Close()

	authStore := auth.NewStore(filepath.Join(*dataDir, "auth.db"))
	authHandler := auth.NewHandler(authStore)

	machineHandler := handler.NewMachineHandler(s)
	sshHandler := handler.NewSSHHandler(s)
	webHandler := handler.NewWebHandler()

	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("/api/login", authHandler.Login)
	mux.HandleFunc("/api/logout", authHandler.Logout)
	mux.HandleFunc("/api/session", authHandler.Session)

	// Machine CRUD API (requires auth)
	mux.HandleFunc("/api/machines", withAuth(authStore, machineHandler.HandleMachines))
	mux.HandleFunc("/api/machines/", withAuth(authStore, machineHandler.HandleMachine))

	// SSH WebSocket proxy (requires auth)
	mux.HandleFunc("/api/ssh/", withAuth(authStore, sshHandler.HandleSSH))

	// Web routes
	mux.HandleFunc("/login", webHandler.LoginPage)
	mux.HandleFunc("/", withAuth(authStore, webHandler.Index))

	// Static files
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Printf("Hermes JumpServer starting on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func withAuth(as *auth.Store, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil || !as.ValidateSession(cookie.Value) {
			if r.Header.Get("Accept") == "application/json" || len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
