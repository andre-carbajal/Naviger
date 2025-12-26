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
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var response NeoForgeVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	var versionsList []string
	seen := make(map[string]bool)

	for _, version := range response.Versions {
		if strings.HasPrefix(version, "0.") || strings.Contains(version, "snapshot") || strings.Contains(version, "alpha") {
			continue
		}

		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			majorNum := parts[0]
			var formatted string

			if majorNum == "20" || majorNum == "21" {
				formatted = fmt.Sprintf("1.%s.%s", majorNum, parts[1])
			} else {
				if len(parts) >= 3 {
					formatted = fmt.Sprintf("%s.%s.%s", majorNum, parts[1], parts[2])
				} else {
					formatted = fmt.Sprintf("%s.%s", majorNum, parts[1])
				}
			}

			if !seen[formatted] {
				versionsList = append(versionsList, formatted)
				seen[formatted] = true
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
		return nil, fmt.Errorf("API responded with status %d", resp.StatusCode)
	}

	var response NeoForgeVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	var loaderVersionsList []string
	parts := strings.Split(minecraftVersion, ".")

	if len(parts) >= 3 {
		// Check if it's old versioning scheme (1.20.x or 1.21.x)
		if parts[0] == "1" && (parts[1] == "20" || parts[1] == "21") {
			// Old scheme: Minecraft 1.21.11 -> NeoForge versions starting with 21.11.
			versionPrefix := parts[1] + "." + parts[2] + "."

			for _, version := range response.Versions {
				if strings.HasPrefix(version, versionPrefix) {
					loaderVersionsList = append(loaderVersionsList, version)
				}
			}
		} else {
			// New scheme: Minecraft 26.1.0 -> NeoForge versions starting with 26.1.0.
			versionPrefix := parts[0] + "." + parts[1] + "." + parts[2] + "."

			for _, version := range response.Versions {
				if strings.HasPrefix(version, versionPrefix) {
					loaderVersionsList = append(loaderVersionsList, version)
				}
			}
		}
	} else if len(parts) == 2 {
		// Handle cases like 1.20 or 26.1
		if parts[0] == "1" && (parts[1] == "20" || parts[1] == "21") {
			// Old scheme: 1.20 -> 20.0.
			versionPrefix := parts[1] + ".0."

			for _, version := range response.Versions {
				if strings.HasPrefix(version, versionPrefix) {
					loaderVersionsList = append(loaderVersionsList, version)
				}
			}
		} else {
			// New scheme: 26.1 -> 26.1.
			versionPrefix := parts[0] + "." + parts[1] + "."

			for _, version := range response.Versions {
				if strings.HasPrefix(version, versionPrefix) {
					loaderVersionsList = append(loaderVersionsList, version)
				}
			}
		}
	}

	SortVersions(loaderVersionsList)
	return loaderVersionsList, nil
}

func (l *NeoForgeLoader) Load(versionID string, destDir string, progressChan chan<- string) error {
	if progressChan != nil {
		progressChan <- fmt.Sprintf("Searching for version %s...", versionID)
	}
	fmt.Printf("[NeoForge Loader] Searching for version %s...\n", versionID)

	supportedVersions, err := l.GetSupportedVersions()
	if err != nil {
		return fmt.Errorf("error getting NeoForge versions: %w", err)
	}

	versionExists := false
	for _, v := range supportedVersions {
		if v == versionID {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return fmt.Errorf("version %s not found in NeoForge", versionID)
	}

	if progressChan != nil {
		progressChan <- "Getting loader versions..."
	}
	loaderVersions, err := l.getLoaderVersions(versionID)
	if err != nil {
		return fmt.Errorf("error getting NeoForge loader versions: %w", err)
	}
	if len(loaderVersions) == 0 {
		return fmt.Errorf("no loader versions found for NeoForge on minecraft version %s", versionID)
	}

	latestLoaderVersion := loaderVersions[0]

	downloadURL := fmt.Sprintf("https://maven.neoforged.net/releases/net/neoforged/neoforge/%s/neoforge-%s-installer.jar", latestLoaderVersion, latestLoaderVersion)

	installerPath := filepath.Join(destDir, "installer.jar")
	if progressChan != nil {
		progressChan <- fmt.Sprintf("Downloading NeoForge installer.jar from: %s", downloadURL)
	}
	fmt.Printf("Downloading NeoForge installer.jar from: %s\n", downloadURL)

	err = l.downloadFile(downloadURL, installerPath)
	if err != nil {
		return err
	}

	if progressChan != nil {
		progressChan <- "Running NeoForge installer..."
	}
	fmt.Println("Running NeoForge installer...")
	cmd := exec.Command("java", "-jar", "installer.jar", "--installServer")
	cmd.Dir = destDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running NeoForge installer: %w", err)
	}

	if progressChan != nil {
		progressChan <- "Cleaning up installation files..."
	}
	fmt.Println("Cleaning up installation files...")
	if err := os.Remove(installerPath); err != nil {
		return fmt.Errorf("error removing installer: %w", err)
	}

	if progressChan != nil {
		progressChan <- "NeoForge installation completed."
	}
	fmt.Println("NeoForge installation completed.")
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
		return fmt.Errorf("error downloading file: status %d", resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}
