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

func (l *VanillaLoader) GetSupportedVersions() ([]string, error) {
	manifest, err := l.fetchManifest()
	if err != nil {
		return nil, fmt.Errorf("could not get version manifest: %w", err)
	}

	var versions []string
	for _, v := range manifest.Versions {
		versions = append(versions, v.ID)
	}

	return versions, nil
}

func (l *VanillaLoader) Load(versionID string, destDir string, progressChan chan<- string) error {
	if progressChan != nil {
		progressChan <- fmt.Sprintf("Searching for version %s...", versionID)
	}
	fmt.Printf("[Vanilla Loader] Searching for version %s...\n", versionID)

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
		return fmt.Errorf("version %s not found in Mojang", versionID)
	}

	if progressChan != nil {
		progressChan <- "Getting version details..."
	}
	details, err := l.fetchVersionDetails(versionURL)
	if err != nil {
		return err
	}

	finalPath := filepath.Join(destDir, "server.jar")
	if progressChan != nil {
		progressChan <- fmt.Sprintf("Downloading server.jar from: %s", details.Downloads.Server.URL)
	}
	fmt.Printf("Downloading server.jar from: %s\n", details.Downloads.Server.URL)

	err = l.downloadFile(details.Downloads.Server.URL, finalPath)
	if err != nil {
		return err
	}

	if progressChan != nil {
		progressChan <- "Installation completed."
	}
	fmt.Println("Installation completed. The server is starting.")
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
