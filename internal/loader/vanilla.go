package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const ManifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

type Manifest struct {
	Versions []Version `json:"versions"`
}
type Version struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
type VersionDetails struct {
	Downloads Downloads `json:"downloads"`
}
type Downloads struct {
	Server DownloadInfo `json:"server"`
}
type DownloadInfo struct {
	URL string `json:"url"`
}

type VanillaLoader struct{}

func NewVanillaLoader() *VanillaLoader {
	return &VanillaLoader{}
}

func (l *VanillaLoader) Load(versionID string, destDir string) error {
	fmt.Printf("[Vanilla Loader] Buscando versión %s...\n", versionID)

	manifest, err := l.fetchManifest()
	if err != nil {
		return err
	}

	var versionURL string
	for _, v := range manifest.Versions {
		if v.ID == versionID {
			versionURL = v.URL
			break
		}
	}
	if versionURL == "" {
		return fmt.Errorf("versión %s no encontrada en Mojang", versionID)
	}

	details, err := l.fetchVersionDetails(versionURL)
	if err != nil {
		return err
	}

	finalPath := filepath.Join(destDir, "server.jar")
	fmt.Printf("Descargando server.jar desde: %s\n", details.Downloads.Server.URL)

	err = l.downloadFile(details.Downloads.Server.URL, finalPath)
	if err != nil {
		return err
	}

	fmt.Println("Instalación completada. El servidor etá iniciando.")
	return nil
}

func (l *VanillaLoader) fetchManifest() (*Manifest, error) {
	resp, err := http.Get(ManifestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var m Manifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (l *VanillaLoader) fetchVersionDetails(url string) (*VersionDetails, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var d VersionDetails
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (l *VanillaLoader) downloadFile(url string, dest string) error {
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
	_, err = io.Copy(out, resp.Body)
	return err
}
