package server

import (
	"fmt"
	"naviger/internal/domain"
	"naviger/internal/loader"
	"naviger/internal/storage"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	ServersPath string
	Store       *storage.GormStore
}

func NewManager(serversPath string, store *storage.GormStore) *Manager {
	return &Manager{
		ServersPath: serversPath,
		Store:       store,
	}
}

func (m *Manager) CreateServer(name string, loaderType string, version string, ram int, progressChan chan<- domain.ProgressEvent) (*domain.Server, error) {
	if strings.ContainsAny(name, "\\/:*?\"<>|") || strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid server name: contains forbidden characters")
	}

	id := uuid.New().String()
	serverDir := filepath.Join(m.ServersPath, id)

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Allocating port..."}
	}
	assignedPort, err := AllocatePort(m.Store)
	if err != nil {
		return nil, fmt.Errorf("error allocating port: %w", err)
	}
	fmt.Printf("Port allocated for '%s': %d\n", name, assignedPort)

	downloader, err := loader.GetLoader(loaderType)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return nil, fmt.Errorf("filesystem error: %w", err)
	}

	if err := downloader.Load(version, serverDir, progressChan); err != nil {
		os.RemoveAll(serverDir)
		return nil, fmt.Errorf("download error: %w", err)
	}

	if progressChan != nil {
		progressChan <- domain.ProgressEvent{Message: "Configuring server..."}
	}
	os.WriteFile(filepath.Join(serverDir, "eula.txt"), []byte("eula=true"), 0644)

	if err := UpdateServerProperties(serverDir, assignedPort); err != nil {
		fmt.Printf("Warning: Could not write server.properties: %v\n", err)
	}

	newServer := &domain.Server{
		ID:        id,
		Name:      name,
		Version:   version,
		Loader:    loaderType,
		Port:      assignedPort,
		RAM:       ram,
		Status:    "STOPPED",
		CreatedAt: time.Now(),
	}

	if err := m.Store.SaveServer(newServer); err != nil {
		os.RemoveAll(serverDir)
		return nil, fmt.Errorf("DB error: %w", err)
	}

	return newServer, nil
}

func (m *Manager) GetServer(id string) (*domain.Server, error) {
	return m.Store.GetServerByID(id)
}

func (m *Manager) ListServers() ([]domain.Server, error) {
	return m.Store.ListServers()
}

func (m *Manager) DeleteServer(id string) error {
	serverDir := filepath.Join(m.ServersPath, id)

	if err := os.RemoveAll(serverDir); err != nil {
		return fmt.Errorf("error deleting server files: %w", err)
	}

	if err := m.Store.DeleteServer(id); err != nil {
		return fmt.Errorf("error deleting server from database: %w", err)
	}

	return nil
}
