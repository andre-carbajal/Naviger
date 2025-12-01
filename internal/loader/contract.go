package loader

type ServerLoader interface {
	Load(version string, destDir string) error
}
