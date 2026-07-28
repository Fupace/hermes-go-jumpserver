package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Fupace/hermes-go-jumpserver/internal/model"
	"github.com/Fupace/hermes-go-jumpserver/internal/store"
)

type MachineHandler struct {
	store *store.Store
}

func NewMachineHandler(s *store.Store) *MachineHandler {
	return &MachineHandler{store: s}
}

func (h *MachineHandler) HandleMachines(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listMachines(w, r)
	case http.MethodPost:
		h.createMachine(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *MachineHandler) HandleMachine(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/machines/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "missing machine ID", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid machine ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getMachine(w, r, id)
	case http.MethodPut:
		h.updateMachine(w, r, id)
	case http.MethodDelete:
		h.deleteMachine(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *MachineHandler) listMachines(w http.ResponseWriter, r *http.Request) {
	machines, err := h.store.ListMachines()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, machines)
}

func (h *MachineHandler) createMachine(w http.ResponseWriter, r *http.Request) {
	var req model.CreateMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Host == "" {
		http.Error(w, "name and host are required", http.StatusBadRequest)
		return
	}
	machine, err := h.store.CreateMachine(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, machine)
}

func (h *MachineHandler) getMachine(w http.ResponseWriter, r *http.Request, id int64) {
	machine, err := h.store.GetMachine(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "machine not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, machine)
}

func (h *MachineHandler) updateMachine(w http.ResponseWriter, r *http.Request, id int64) {
	var req model.UpdateMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	machine, err := h.store.UpdateMachine(id, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "machine not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, machine)
}

func (h *MachineHandler) deleteMachine(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.store.DeleteMachine(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
