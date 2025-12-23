package loader

type ServerLoader interface {
	Load(version string, destDir string, progressChan chan<- string) error
	GetSupportedVersions() ([]string, error)
}
