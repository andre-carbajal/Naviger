package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"mc-manager/internal/storage"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Manager struct {
	ServersPath string
	BackupsPath string
	Store       *storage.SQLiteStore
}

func NewManager(serversPath, backupsPath string, store *storage.SQLiteStore) *Manager {
	return &Manager{
		ServersPath: serversPath,
		BackupsPath: backupsPath,
		Store:       store,
	}
}

func (m *Manager) CreateBackup(serverID string, backupName string) (string, error) {
	serverDir := filepath.Join(m.ServersPath, serverID)
	if _, err := os.Stat(serverDir); os.IsNotExist(err) {
		return "", fmt.Errorf("el directorio del servidor con ID '%s' no existe", serverID)
	}

	if backupName == "" {
		srv, err := m.Store.GetServerByID(serverID)
		if err != nil {
			return "", fmt.Errorf("no se pudo obtener la información del servidor: %w", err)
		}
		if srv == nil {
			return "", fmt.Errorf("servidor con ID '%s' no encontrado en la base de datos", serverID)
		}
		backupName = srv.Name
	}

	safeName := sanitizeFileName(backupName)
	timestamp := time.Now().Format("20060102-150405")
	backupFileName := fmt.Sprintf("%s-%s.zip", safeName, timestamp)
	backupFilePath := filepath.Join(m.BackupsPath, backupFileName)

	if err := os.MkdirAll(m.BackupsPath, 0755); err != nil {
		return "", fmt.Errorf("no se pudo crear el directorio de backups: %w", err)
	}

	backupFile, err := os.Create(backupFilePath)
	if err != nil {
		return "", fmt.Errorf("no se pudo crear el archivo de backup: %w", err)
	}
	defer backupFile.Close()

	zipWriter := zip.NewWriter(backupFile)
	defer zipWriter.Close()

	err = filepath.Walk(serverDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(serverDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
		}
		return err
	})

	if err != nil {
		os.Remove(backupFilePath)
		return "", fmt.Errorf("error al crear el backup: %w", err)
	}

	return backupFilePath, nil
}

func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, " ", "-")
	reg := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	sanitized := reg.ReplaceAllString(name, "")
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}
	return sanitized
}
