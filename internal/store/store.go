package store

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Fupace/hermes-go-jumpserver/internal/model"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS machines (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		host TEXT NOT NULL,
		port INTEGER NOT NULL DEFAULT 22,
		username TEXT NOT NULL DEFAULT 'root',
		password TEXT NOT NULL DEFAULT '',
		key_file TEXT NOT NULL DEFAULT '',
		labels TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.Exec(query)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) ListMachines() ([]model.Machine, error) {
	rows, err := s.db.Query(`SELECT id, name, host, port, username, password, key_file, labels, created_at, updated_at FROM machines ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var machines []model.Machine
	for rows.Next() {
		var m model.Machine
		if err := rows.Scan(&m.ID, &m.Name, &m.Host, &m.Port, &m.Username, &m.Password, &m.KeyFile, &m.Labels, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		machines = append(machines, m)
	}
	if machines == nil {
		machines = []model.Machine{}
	}
	return machines, rows.Err()
}

func (s *Store) GetMachine(id int64) (*model.Machine, error) {
	var m model.Machine
	err := s.db.QueryRow(`SELECT id, name, host, port, username, password, key_file, labels, created_at, updated_at FROM machines WHERE id = ?`, id).
		Scan(&m.ID, &m.Name, &m.Host, &m.Port, &m.Username, &m.Password, &m.KeyFile, &m.Labels, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) CreateMachine(req *model.CreateMachineRequest) (*model.Machine, error) {
	now := time.Now()
	port := req.Port
	if port == 0 {
		port = 22
	}
	result, err := s.db.Exec(
		`INSERT INTO machines (name, host, port, username, password, key_file, labels, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Name, req.Host, port, req.Username, req.Password, req.KeyFile, req.Labels, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	m := &model.Machine{
		ID:        id,
		Name:      req.Name,
		Host:      req.Host,
		Port:      port,
		Username:  req.Username,
		Password:  req.Password,
		KeyFile:   req.KeyFile,
		Labels:    req.Labels,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return m, nil
}

func (s *Store) UpdateMachine(id int64, req *model.UpdateMachineRequest) (*model.Machine, error) {
	existing, err := s.GetMachine(id)
	if err != nil || existing == nil {
		return nil, err
	}

	name := existing.Name
	host := existing.Host
	port := existing.Port
	username := existing.Username
	password := existing.Password
	keyFile := existing.KeyFile
	labels := existing.Labels

	if req.Name != nil {
		name = *req.Name
	}
	if req.Host != nil {
		host = *req.Host
	}
	if req.Port != nil {
		port = *req.Port
	}
	if req.Username != nil {
		username = *req.Username
	}
	if req.Password != nil {
		password = *req.Password
	}
	if req.KeyFile != nil {
		keyFile = *req.KeyFile
	}
	if req.Labels != nil {
		labels = *req.Labels
	}

	now := time.Now()
	_, err = s.db.Exec(
		`UPDATE machines SET name=?, host=?, port=?, username=?, password=?, key_file=?, labels=?, updated_at=? WHERE id=?`,
		name, host, port, username, password, keyFile, labels, now, id,
	)
	if err != nil {
		return nil, err
	}

	m := &model.Machine{
		ID:        id,
		Name:      name,
		Host:      host,
		Port:      port,
		Username:  username,
		Password:  password,
		KeyFile:   keyFile,
		Labels:    labels,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: now,
	}
	return m, nil
}

func (s *Store) DeleteMachine(id int64) error {
	_, err := s.db.Exec(`DELETE FROM machines WHERE id = ?`, id)
	return err
}
