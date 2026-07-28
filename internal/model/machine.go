package model

import "time"

type Machine struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Username  string    `json:"username"`
	Password  string    `json:"password,omitempty"`
	KeyFile   string    `json:"key_file,omitempty"`
	Labels    string    `json:"labels"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateMachineRequest struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	KeyFile  string `json:"key_file"`
	Labels   string `json:"labels"`
}

type UpdateMachineRequest struct {
	Name     *string `json:"name,omitempty"`
	Host     *string `json:"host,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
	KeyFile  *string `json:"key_file,omitempty"`
	Labels   *string `json:"labels,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token   string `json:"token"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}
