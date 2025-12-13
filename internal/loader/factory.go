package loader

import "fmt"

func GetLoader(loaderType string) (ServerLoader, error) {
	switch loaderType {
	case "vanilla":
		return NewVanillaLoader(), nil
	case "paper":
		return NewPaperLoader(), nil
	case "fabric":
		return NewFabricLoader(), nil
	default:
		return nil, fmt.Errorf("tipo de loader '%s' no soportado", loaderType)
	}
}

func GetLoaderVersions(loaderType string) ([]string, error) {
	loader, err := GetLoader(loaderType)
	if err != nil {
		return nil, err
	}
	return loader.GetSupportedVersions()
}

func GetAvailableLoaders() []string {
	return []string{"vanilla", "paper", "fabric"}
}
