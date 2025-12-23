package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const NeoForgeAPIURL = "https://maven.neoforged.net/api/maven/versions/releases/net%2Fneoforged%2Fneoforge"

type NeoForgeLoader struct{}

func NewNeoForgeLoader() *NeoForgeLoader {
	return &NeoForgeLoader{}
}

type NeoForgeVersionsResponse struct {
	Versions []string `json:"versions"`
}

func (l *NeoForgeLoader) GetSupportedVersions() ([]string, error) {
	resp, err := http.Get(NeoForgeAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API respondió con status %d", resp.StatusCode)
	}

	var response NeoForgeVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	var versionsList []string
	seen := make(map[string]bool)

	for _, version := range response.Versions {
		if !strings.HasPrefix(version, "0.") {
			parts := strings.Split(version, ".")
			if len(parts) >= 2 {
				formatted := fmt.Sprintf("1.%s.%s", parts[0], parts[1])

				if !seen[formatted] {
					versionsList = append(versionsList, formatted)
					seen[formatted] = true
				}
			}
		}
	}

	SortVersions(versionsList)
	return versionsList, nil
}

func (l *NeoForgeLoader) getLoaderVersions(minecraftVersion string) ([]string, error) {
	resp, err := http.Get(NeoForgeAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API respondió con status %d", resp.StatusCode)
	}

	var response NeoForgeVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	var loaderVersionsList []string
	parts := strings.Split(minecraftVersion, ".")

	if len(parts) >= 3 {
		versionPrefix := parts[1] + "." + parts[2] + "."

		for _, version := range response.Versions {
			if strings.HasPrefix(version, versionPrefix) {
				loaderVersionsList = append(loaderVersionsList, version)
			}
		}
	}

	SortVersions(loaderVersionsList)
	return loaderVersionsList, nil
}

func (l *NeoForgeLoader) Load(versionID string, destDir string, progressChan chan<- string) error {
	if progressChan != nil {
		progressChan <- fmt.Sprintf("Buscando versión %s...", versionID)
	}
	fmt.Printf("[NeoForge Loader] Buscando versión %s...\n", versionID)

	supportedVersions, err := l.GetSupportedVersions()
	if err != nil {
		return fmt.Errorf("error obteniendo versiones de NeoForge: %w", err)
	}

	versionExists := false
	for _, v := range supportedVersions {
		if v == versionID {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return fmt.Errorf("versión %s no encontrada en NeoForge", versionID)
	}

	if progressChan != nil {
		progressChan <- "Obteniendo versiones del loader..."
	}
	loaderVersions, err := l.getLoaderVersions(versionID)
	if err != nil {
		return fmt.Errorf("error obteniendo versiones del loader de NeoForge: %w", err)
	}
	if len(loaderVersions) == 0 {
		return fmt.Errorf("no se encontraron versiones del loader para NeoForge en la version de minecraft %s", versionID)
	}

	latestLoaderVersion := loaderVersions[0]

	downloadURL := fmt.Sprintf("https://maven.neoforged.net/releases/net/neoforged/neoforge/%s/neoforge-%s-installer.jar", latestLoaderVersion, latestLoaderVersion)

	installerPath := filepath.Join(destDir, "installer.jar")
	if progressChan != nil {
		progressChan <- fmt.Sprintf("Descargando NeoForge installer.jar desde: %s", downloadURL)
	}
	fmt.Printf("Descargando NeoForge installer.jar desde: %s\n", downloadURL)

	err = l.downloadFile(downloadURL, installerPath)
	if err != nil {
		return err
	}

	if progressChan != nil {
		progressChan <- "Ejecutando instalador de NeoForge..."
	}
	fmt.Println("Ejecutando instalador de NeoForge...")
	cmd := exec.Command("java", "-jar", "installer.jar", "--installServer")
	cmd.Dir = destDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error ejecutando el instalador de NeoForge: %w", err)
	}

	if progressChan != nil {
		progressChan <- "Limpiando archivos de instalación..."
	}
	fmt.Println("Limpiando archivos de instalación...")
	if err := os.Remove(installerPath); err != nil {
		return fmt.Errorf("error eliminando el instalador: %w", err)
	}

	if progressChan != nil {
		progressChan <- "Instalación de NeoForge completada."
	}
	fmt.Println("Instalación de NeoForge completada.")
	return nil
}

func (l *NeoForgeLoader) downloadFile(url string, dest string) error {
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
