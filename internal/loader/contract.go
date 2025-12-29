package loader

import "naviger/internal/domain"

type ServerLoader interface {
	Load(version string, destDir string, progressChan chan<- domain.ProgressEvent) error
	GetSupportedVersions() ([]string, error)
}
