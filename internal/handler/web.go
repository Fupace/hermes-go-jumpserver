package handler

import (
	"embed"
	"net/http"
)

//go:embed templates/*
var templates embed.FS

type WebHandler struct{}

func NewWebHandler() *WebHandler {
	return &WebHandler{}
}

func (h *WebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	data, err := templates.ReadFile("templates/login.html")
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	data, err := templates.ReadFile("templates/index.html")
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
