package storage

import (
	"errors"
	"fmt"
	"log"
	"naviger/internal/domain"
	"os"
	"strconv"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Server struct {
	ID         string `gorm:"primaryKey"`
	Name       string
	FolderName string
	Version    string
	Loader     string
	Port       int
	RAM        int
	Status     string
	CustomArgs string
	CreatedAt  time.Time
}

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

type GormStore struct {
	db *gorm.DB
}

func NewGormStore(path string) (*GormStore, error) {
	newLogger := gormlogger.New(
		log.New(os.Stdout, "", log.LstdFlags),
		gormlogger.Config{
			IgnoreRecordNotFoundError: true,
			LogLevel:                  gormlogger.Error,
		},
	)

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: newLogger})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Server{}, &Setting{})
	if err != nil {
		return nil, fmt.Errorf("error migrating database: %w", err)
	}

	store := &GormStore{db: db}

	if err := store.initDefaultSettings(); err != nil {
		return nil, fmt.Errorf("error initializing settings: %w", err)
	}

	return store, nil
}

func (s *GormStore) initDefaultSettings() error {
	defaults := map[string]string{
		"port_range_start": "25565",
		"port_range_end":   "25600",
	}

	for key, value := range defaults {
		var setting Setting
		result := s.db.First(&setting, "key = ?", key)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				if err := s.db.Create(&Setting{Key: key, Value: value}).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		}
	}

	return nil
}

func (s *GormStore) SaveServer(srv *domain.Server) error {
	gormServer := &Server{
		ID:         srv.ID,
		Name:       srv.Name,
		FolderName: srv.FolderName,
		Version:    srv.Version,
		Loader:     srv.Loader,
		Port:       srv.Port,
		RAM:        srv.RAM,
		Status:     srv.Status,
		CustomArgs: srv.CustomArgs,
		CreatedAt:  srv.CreatedAt,
	}

	return s.db.Create(gormServer).Error
}

func (s *GormStore) UpdateServer(id string, name *string, ram *int, customArgs *string) error {
	if name == nil && ram == nil && customArgs == nil {
		return errors.New("no fields to update")
	}

	updates := make(map[string]interface{})
	if name != nil {
		updates["name"] = *name
	}
	if ram != nil {
		updates["ram"] = *ram
	}
	if customArgs != nil {
		updates["custom_args"] = *customArgs
	}

	return s.db.Model(&Server{}).Where("id = ?", id).Updates(updates).Error
}

func (s *GormStore) UpdateServerPort(id string, port int) error {
	return s.db.Model(&Server{}).Where("id = ?", id).Update("port", port).Error
}

func (s *GormStore) ListServers() ([]domain.Server, error) {
	var gormServers []Server
	if err := s.db.Find(&gormServers).Error; err != nil {
		return nil, err
	}

	var servers []domain.Server
	for _, gs := range gormServers {
		servers = append(servers, domain.Server{
			ID:         gs.ID,
			Name:       gs.Name,
			FolderName: gs.FolderName,
			Version:    gs.Version,
			Loader:     gs.Loader,
			Port:       gs.Port,
			RAM:        gs.RAM,
			Status:     gs.Status,
			CustomArgs: gs.CustomArgs,
			CreatedAt:  gs.CreatedAt,
		})
	}
	return servers, nil
}

func (s *GormStore) GetServerByID(id string) (*domain.Server, error) {
	var gormServer Server
	result := s.db.First(&gormServer, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("error querying server: %w", result.Error)
	}

	return &domain.Server{
		ID:         gormServer.ID,
		Name:       gormServer.Name,
		FolderName: gormServer.FolderName,
		Version:    gormServer.Version,
		Loader:     gormServer.Loader,
		Port:       gormServer.Port,
		RAM:        gormServer.RAM,
		Status:     gormServer.Status,
		CustomArgs: gormServer.CustomArgs,
		CreatedAt:  gormServer.CreatedAt,
	}, nil
}

func (s *GormStore) DeleteServer(id string) error {
	return s.db.Delete(&Server{}, "id = ?", id).Error
}

func (s *GormStore) UpdateStatus(id string, status string) error {
	return s.db.Model(&Server{}).Where("id = ?", id).Update("status", status).Error
}

func (s *GormStore) GetSetting(key string) (string, error) {
	var setting Setting
	result := s.db.First(&setting, "key = ?", key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("setting not found: %s", key)
		}
		return "", result.Error
	}
	return setting.Value, nil
}

func (s *GormStore) SetSetting(key string, value string) error {
	var setting Setting
	result := s.db.First(&setting, "key = ?", key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return s.db.Create(&Setting{Key: key, Value: value}).Error
		}
		return result.Error
	}

	return s.db.Model(&setting).Update("value", value).Error
}

func (s *GormStore) GetPortRange() (int, int, error) {
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
		return 0, 0, fmt.Errorf("error parsing port_range_start: %w", err)
	}

	end, err := strconv.Atoi(endStr)
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing port_range_end: %w", err)
	}

	return start, end, nil
}

func (s *GormStore) SetPortRange(start int, end int) error {
	if start <= 0 || end <= 0 || start > end {
		return fmt.Errorf("invalid port range: %d-%d", start, end)
	}

	if err := s.SetSetting("port_range_start", fmt.Sprintf("%d", start)); err != nil {
		return err
	}

	if err := s.SetSetting("port_range_end", fmt.Sprintf("%d", end)); err != nil {
		return err
	}

	return nil
}
