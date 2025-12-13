package server

import (
	"fmt"
	"mc-manager/internal/domain"
	"mc-manager/internal/loader"
	"mc-manager/internal/storage"
	"os"
	"path/filepath"
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

func (m *Manager) CreateServer(name string, loaderType string, version string, ram int) (*domain.Server, error) {
	id := uuid.New().String()
	serverDir := filepath.Join(m.ServersPath, id)

	assignedPort, err := AllocatePort(m.Store)
	if err != nil {
		return nil, fmt.Errorf("error asignando puerto: %w", err)
	}
	fmt.Printf("Puerto asignado para '%s': %d\n", name, assignedPort)

	downloader, err := loader.GetLoader(loaderType)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return nil, fmt.Errorf("error filesystem: %w", err)
	}

	if err := downloader.Load(version, serverDir); err != nil {
		os.RemoveAll(serverDir)
		return nil, fmt.Errorf("error descarga: %w", err)
	}

	os.WriteFile(filepath.Join(serverDir, "eula.txt"), []byte("eula=true"), 0644)

	if err := UpdateServerProperties(serverDir, assignedPort); err != nil {
		fmt.Printf("Advertencia: No se pudo escribir server.properties: %v\n", err)
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
		return nil, fmt.Errorf("error DB: %w", err)
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
		return fmt.Errorf("error eliminando archivos del servidor: %w", err)
	}

	if err := m.Store.DeleteServer(id); err != nil {
		return fmt.Errorf("error eliminando servidor de la base de datos: %w", err)
	}

	return nil
}
