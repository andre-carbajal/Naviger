package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const ForgeAPIURL = "https://bmclapi2.bangbang93.com/forge/"

type ForgeLoader struct{}

func NewForgeLoader() *ForgeLoader {
	return &ForgeLoader{}
}

func (l *ForgeLoader) GetSupportedVersions() ([]string, error) {
	resp, err := http.Get(ForgeAPIURL + "minecraft")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API respondió con status %d", resp.StatusCode)
	}

	var versions []string
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	SortVersions(versions)
	return versions, nil
}

func (l *ForgeLoader) getLoaderVersions(minecraftVersion string) ([]string, error) {
	resp, err := http.Get(ForgeAPIURL + "minecraft/" + minecraftVersion)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API respondió con status %d", resp.StatusCode)
	}

	type forgeLoaderVersion struct {
		Version string `json:"version"`
	}

	var loaderInfo []forgeLoaderVersion
	if err := json.NewDecoder(resp.Body).Decode(&loaderInfo); err != nil {
		return nil, err
	}

	var versions []string
	for _, v := range loaderInfo {
		versions = append(versions, v.Version)
	}

	SortVersions(versions)
	return versions, nil
}

func (l *ForgeLoader) Load(versionID string, destDir string, progressChan chan<- string) error {
	if progressChan != nil {
		progressChan <- fmt.Sprintf("Buscando versión %s...", versionID)
	}
	fmt.Printf("[Forge Loader] Buscando versión %s...\n", versionID)

	supportedVersions, err := l.GetSupportedVersions()
	if err != nil {
		return fmt.Errorf("error obteniendo versiones de Forge: %w", err)
	}

	versionExists := false
	for _, v := range supportedVersions {
		if v == versionID {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return fmt.Errorf("versión %s no encontrada en Forge", versionID)
	}

	if progressChan != nil {
		progressChan <- "Obteniendo versiones del loader..."
	}
	loaderVersions, err := l.getLoaderVersions(versionID)
	if err != nil {
		return fmt.Errorf("error obteniendo versiones del loader de Forge: %w", err)
	}
	if len(loaderVersions) == 0 {
		return fmt.Errorf("no se encontraron versiones del loader para Forge en la version de minecraft %s", versionID)
	}
	latestLoaderVersion := loaderVersions[0]

	forgeVersion := fmt.Sprintf("%s-%s", versionID, latestLoaderVersion)
	downloadURL := fmt.Sprintf("https://maven.minecraftforge.net/net/minecraftforge/forge/%s/forge-%s-installer.jar", forgeVersion, forgeVersion)

	installerPath := filepath.Join(destDir, "installer.jar")
	if progressChan != nil {
		progressChan <- fmt.Sprintf("Descargando Forge installer.jar desde: %s", downloadURL)
	}
	fmt.Printf("Descargando Forge installer.jar desde: %s\n", downloadURL)

	err = l.downloadFile(downloadURL, installerPath)
	if err != nil {
		return err
	}

	if progressChan != nil {
		progressChan <- "Ejecutando instalador de Forge..."
	}
	fmt.Println("Ejecutando instalador de Forge...")
	cmd := exec.Command("java", "-jar", "installer.jar", "--installServer")
	cmd.Dir = destDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error ejecutando el instalador de Forge: %w", err)
	}

	if progressChan != nil {
		progressChan <- "Limpiando archivos de instalación..."
	}
	fmt.Println("Limpiando archivos de instalación...")
	if err := os.Remove(installerPath); err != nil {
		return fmt.Errorf("error eliminando el instalador: %w", err)
	}

	if progressChan != nil {
		progressChan <- "Instalación de Forge completada."
	}
	fmt.Println("Instalación de Forge completada.")
	return nil
}

func (l *ForgeLoader) downloadFile(url string, dest string) error {
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
