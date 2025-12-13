package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const FabricAPIURL = "https://meta.fabricmc.net/v2/versions/"

type FabricGameVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

type FabricLoaderVersion struct {
	Version string `json:"version"`
}

type FabricInstallerVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

type FabricLoader struct{}

func NewFabricLoader() *FabricLoader {
	return &FabricLoader{}
}

func (l *FabricLoader) GetSupportedVersions() ([]string, error) {
	return l.getGameVersions()
}

func (l *FabricLoader) Load(versionID string, destDir string) error {
	fmt.Printf("[Fabric Loader] Buscando versión %s...\n", versionID)

	gameVersions, err := l.getGameVersions()
	if err != nil {
		return fmt.Errorf("error obteniendo versiones de Fabric: %w", err)
	}

	versionExists := false
	for _, v := range gameVersions {
		if v == versionID {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return fmt.Errorf("versión %s no encontrada en Fabric", versionID)
	}

	loaderVersions, err := l.getLoaderVersions()
	if err != nil {
		return fmt.Errorf("error obteniendo versiones del loader de Fabric: %w", err)
	}
	if len(loaderVersions) == 0 {
		return fmt.Errorf("no se encontraron versiones del loader para Fabric")
	}
	latestLoaderVersion := loaderVersions[0]

	installerVersion, err := l.getLatestInstallerVersion()
	if err != nil {
		return fmt.Errorf("error obteniendo la última versión del instalador: %w", err)
	}

	downloadURL := fmt.Sprintf("%sloader/%s/%s/%s/server/jar",
		FabricAPIURL, versionID, latestLoaderVersion, installerVersion)

	finalPath := filepath.Join(destDir, "server.jar")
	fmt.Printf("Descargando Fabric server.jar desde: %s\n", downloadURL)

	err = l.downloadFile(downloadURL, finalPath)
	if err != nil {
		return err
	}

	fmt.Println("Instalación completada. El servidor está iniciando.")
	return nil
}

func (l *FabricLoader) getGameVersions() ([]string, error) {
	resp, err := http.Get(FabricAPIURL + "game")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API respondió con status %d", resp.StatusCode)
	}

	var versions []FabricGameVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	var stableVersions []string
	for _, v := range versions {
		if v.Stable {
			stableVersions = append(stableVersions, v.Version)
		}
	}

	return stableVersions, nil
}

func (l *FabricLoader) getLoaderVersions() ([]string, error) {
	resp, err := http.Get(FabricAPIURL + "loader")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API respondió con status %d", resp.StatusCode)
	}

	var versions []FabricLoaderVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	var loaderVersions []string
	for _, v := range versions {
		loaderVersions = append(loaderVersions, v.Version)
	}

	return loaderVersions, nil
}

func (l *FabricLoader) getLatestInstallerVersion() (string, error) {
	resp, err := http.Get(FabricAPIURL + "installer")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API respondió con status %d", resp.StatusCode)
	}

	var versions []FabricInstallerVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return "", err
	}

	for _, v := range versions {
		if v.Stable {
			return v.Version, nil
		}
	}

	return "", fmt.Errorf("no se encontró una versión estable del instalador")
}

func (l *FabricLoader) downloadFile(url string, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error descargando archivo: status %d", resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}
