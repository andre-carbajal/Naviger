package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"mc-manager/internal/domain"
	"strconv"
	"strings"

	_ "github.com/glebarez/go-sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	query := `
	CREATE TABLE IF NOT EXISTS servers (
		id TEXT PRIMARY KEY,
		name TEXT,
		version TEXT,
		loader TEXT,
		port INTEGER,
		ram INTEGER,
		status TEXT,
		created_at DATETIME
	);
	
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);`

	_, err = db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("error migrando la base de datos: %w", err)
	}

	store := &SQLiteStore{db: db}

	if err := store.initDefaultSettings(); err != nil {
		return nil, fmt.Errorf("error inicializando configuración: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) initDefaultSettings() error {
	defaults := map[string]string{
		"port_range_start": "25565",
		"port_range_end":   "25600",
	}

	for key, value := range defaults {
		var exists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM settings WHERE key = ?)", key).Scan(&exists)
		if err != nil {
			return err
		}

		if !exists {
			_, err = s.db.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", key, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SQLiteStore) SaveServer(srv *domain.Server) error {
	query := `
	INSERT INTO servers (id, name, version, loader, port, ram, status, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, srv.ID, srv.Name, srv.Version, srv.Loader, srv.Port, srv.RAM, srv.Status, srv.CreatedAt)
	return err
}

func (s *SQLiteStore) UpdateServer(id string, name *string, ram *int) error {
	if name == nil && ram == nil {
		return errors.New("no fields to update")
	}

	var parts []string
	var args []interface{}

	if name != nil {
		parts = append(parts, "name = ?")
		args = append(args, *name)
	}
	if ram != nil {
		parts = append(parts, "ram = ?")
		args = append(args, *ram)
	}

	query := fmt.Sprintf("UPDATE servers SET %s WHERE id = ?", strings.Join(parts, ", "))
	args = append(args, id)

	_, err := s.db.Exec(query, args...)
	return err
}

func (s *SQLiteStore) ListServers() ([]domain.Server, error) {
	rows, err := s.db.Query("SELECT id, name, version, loader, port, ram, status, created_at FROM servers")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var servers []domain.Server
	for rows.Next() {
		var srv domain.Server
		if err := rows.Scan(&srv.ID, &srv.Name, &srv.Version, &srv.Loader, &srv.Port, &srv.RAM, &srv.Status, &srv.CreatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, srv)
	}
	return servers, nil
}

func (s *SQLiteStore) GetServerByID(id string) (*domain.Server, error) {
	row := s.db.QueryRow("SELECT id, name, version, loader, port, ram, status, created_at FROM servers WHERE id = ?", id)
	var srv domain.Server
	if err := row.Scan(&srv.ID, &srv.Name, &srv.Version, &srv.Loader, &srv.Port, &srv.RAM, &srv.Status, &srv.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error consultando server: %w", err)
	}
	return &srv, nil
}

func (s *SQLiteStore) DeleteServer(id string) error {
	_, err := s.db.Exec("DELETE FROM servers WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) UpdateStatus(id string, status string) error {
	_, err := s.db.Exec("UPDATE servers SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return fmt.Errorf("error actualizando estado: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("configuración no encontrada: %s", key)
		}
		return "", err
	}
	return value, nil
}

func (s *SQLiteStore) SetSetting(key string, value string) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

func (s *SQLiteStore) GetPortRange() (int, int, error) {
	startStr, err := s.GetSetting("port_range_start")
	if err != nil {
		return 0, 0, err
	}

	endStr, err := s.GetSetting("port_range_end")
	if err != nil {
		return 0, 0, err
	}

	start, err := strconv.Atoi(startStr)
	if err != nil {
		return 0, 0, fmt.Errorf("error parseando port_range_start: %w", err)
	}

	end, err := strconv.Atoi(endStr)
	if err != nil {
		return 0, 0, fmt.Errorf("error parseando port_range_end: %w", err)
	}

	return start, end, nil
}

func (s *SQLiteStore) SetPortRange(start int, end int) error {
	if start <= 0 || end <= 0 || start > end {
		return fmt.Errorf("rango de puertos inválido: %d-%d", start, end)
	}

	if err := s.SetSetting("port_range_start", fmt.Sprintf("%d", start)); err != nil {
		return err
	}

	if err := s.SetSetting("port_range_end", fmt.Sprintf("%d", end)); err != nil {
		return err
	}

	return nil
}
