package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const PaperAPIURL = "https://api.papermc.io/v2/projects/paper/"

type PaperVersionsResponse struct {
	Versions []string `json:"versions"`
}

type PaperBuildsResponse struct {
	Builds []int `json:"builds"`
}

type PaperLoader struct{}

func NewPaperLoader() *PaperLoader {
	return &PaperLoader{}
}

func (l *PaperLoader) Load(versionID string, destDir string) error {
	fmt.Printf("[Paper Loader] Buscando versión %s...\n", versionID)

	versions, err := l.getVersions()
	if err != nil {
		return fmt.Errorf("error obteniendo versiones de Paper: %w", err)
	}

	versionExists := false
	for _, v := range versions {
		if v == versionID {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return fmt.Errorf("versión %s no encontrada en Paper", versionID)
	}

	latestBuild, err := l.getLatestBuild(versionID)
	if err != nil {
		return fmt.Errorf("error obteniendo último build: %w", err)
	}

	downloadURL := fmt.Sprintf("%sversions/%s/builds/%d/downloads/paper-%s-%d.jar",
		PaperAPIURL, versionID, latestBuild, versionID, latestBuild)

	finalPath := filepath.Join(destDir, "server.jar")
	fmt.Printf("Descargando Paper server.jar desde: %s\n", downloadURL)

	err = l.downloadFile(downloadURL, finalPath)
	if err != nil {
		return err
	}

	fmt.Println("Instalación completada. El servidor etá iniciando.")
	return nil
}

func (l *PaperLoader) getVersions() ([]string, error) {
	resp, err := http.Get(PaperAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API respondió con status %d", resp.StatusCode)
	}

	var response PaperVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	var filteredVersions []string
	for _, v := range response.Versions {
		if !strings.Contains(v, "-") {
			filteredVersions = append(filteredVersions, v)
		}
	}

	return filteredVersions, nil
}

func (l *PaperLoader) getLatestBuild(version string) (int, error) {
	url := fmt.Sprintf("%sversions/%s", PaperAPIURL, version)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API respondió con status %d", resp.StatusCode)
	}

	var response PaperBuildsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	if len(response.Builds) == 0 {
		return 0, fmt.Errorf("no se encontraron builds para la versión %s", version)
	}

	return response.Builds[len(response.Builds)-1], nil
}

func (l *PaperLoader) downloadFile(url string, dest string) error {
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
