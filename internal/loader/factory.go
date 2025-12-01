package loader

import "fmt"

func GetLoader(loaderType string) (ServerLoader, error) {
	switch loaderType {
	case "vanilla":
		return NewVanillaLoader(), nil
	case "paper":
		return NewPaperLoader(), nil
	default:
		return nil, fmt.Errorf("tipo de loader '%s' no soportado", loaderType)
	}
}
