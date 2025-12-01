package jvm

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type Manager struct {
	RuntimesPath string
}

func NewManager(runtimesPath string) *Manager {
	return &Manager{RuntimesPath: runtimesPath}
}

func (m *Manager) EnsureJava(version int) (string, error) {
	installDir := filepath.Join(m.RuntimesPath, fmt.Sprintf("java-%d", version))

	javaBinName := "java"
	if runtime.GOOS == "windows" {
		javaBinName = "java.exe"
	}

	if !isTrueEnv("MC_MANAGER_FORCE_DOWNLOAD") {
		if p := os.Getenv("MC_MANAGER_JAVA_PATH"); p != "" {
			candidate := p
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				candidate = filepath.Join(candidate, "bin", javaBinName)
			}
			if _, err := os.Stat(candidate); err == nil {
				if ok, _ := validateJavaVersion(candidate, version); ok {
					abs, err := filepath.Abs(candidate)
					if err == nil {
						return abs, nil
					}
				}
			}
		}

		if jh := os.Getenv("JAVA_HOME"); jh != "" {
			candidate := filepath.Join(jh, "bin", javaBinName)
			if _, err := os.Stat(candidate); err == nil {
				if ok, _ := validateJavaVersion(candidate, version); ok {
					abs, err := filepath.Abs(candidate)
					if err == nil {
						return abs, nil
					}
				}
			}
		}

		if fi, err := os.Stat(installDir); err == nil && fi.IsDir() {
			if found, err := findJavaBin(installDir, javaBinName); err == nil {
				if ok, _ := validateJavaVersion(found, version); ok {
					abs, err := filepath.Abs(found)
					if err == nil {
						return abs, nil
					}
				}
			}
		}
	}

	javaExec := filepath.Join(installDir, "bin", javaBinName)
	if _, err := os.Stat(javaExec); err == nil {
		if ok, _ := validateJavaVersion(javaExec, version); ok {
			return javaExec, nil
		}
	}

	fmt.Printf("☕ Java %d no detectado. Iniciando instalación automática (%s)...\n", version, runtime.GOOS)

	if err := m.downloadAndInstall(version, installDir); err != nil {
		_ = os.RemoveAll(installDir)
		return "", err
	}

	finalBin, err := findJavaBin(installDir, javaBinName)
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(finalBin)
	if err != nil {
		return "", fmt.Errorf("no se pudo obtener ruta absoluta: %w", err)
	}

	if runtime.GOOS != "windows" {
		_ = os.Chmod(absPath, 0755)
	}

	return absPath, nil
}

func (m *Manager) downloadAndInstall(version int, destDir string) error {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	var ext string
	var apiOS string

	switch osName {
	case "windows":
		apiOS = "windows"
		ext = ".zip"
	case "darwin":
		apiOS = "mac"
		ext = ".tar.gz"
	case "linux":
		apiOS = "linux"
		ext = ".tar.gz"
	default:
		return fmt.Errorf("sistema operativo no soportado: %s", osName)
	}

	switch arch {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "aarch64"
	default:
		return fmt.Errorf("arquitectura no soportada: %s", arch)
	}

	url := fmt.Sprintf(
		"https://api.adoptium.net/v3/binary/latest/%d/ga/%s/%s/jre/hotspot/normal/eclipse",
		version, apiOS, arch,
	)

	fmt.Printf("Descargando JRE desde: %s\n", url)

	tmpFile, err := os.CreateTemp("", "jdk-*"+ext)
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer func(p string) { _ = os.Remove(p) }(tmpPath)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error de red: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return fmt.Errorf("API Adoptium error: %d", resp.StatusCode)
	}

	copyErr := func() error {
		_, err := io.Copy(tmpFile, resp.Body)
		if err != nil {
			_ = tmpFile.Close()
			return err
		}
		if closeErr := tmpFile.Close(); closeErr != nil {
			return fmt.Errorf("error cerrando archivo temporal: %w", closeErr)
		}
		return nil
	}()
	if copyErr != nil {
		return copyErr
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	fmt.Printf("Descomprimiendo %s...\n", ext)

	if ext == ".zip" {
		if err := Unzip(tmpPath, destDir); err != nil {
			return fmt.Errorf("error unzip: %w", err)
		}
	} else {
		if err := Untar(tmpPath, destDir); err != nil {
			return fmt.Errorf("error untar: %w", err)
		}
	}

	return nil
}

func findJavaBin(root, binName string) (string, error) {
	var foundPath string
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == binName {
			if info.Mode()&0111 != 0 || runtime.GOOS == "windows" {
				foundPath = path
				return io.EOF
			}
		}
		return nil
	})

	if walkErr != nil && walkErr != io.EOF {
		return "", fmt.Errorf("error recorriendo %s: %w", root, walkErr)
	}

	if foundPath != "" {
		return foundPath, nil
	}
	return "", fmt.Errorf("binario %s no encontrado tras instalación", binName)
}

func validateJavaVersion(javaPath string, required int) (bool, error) {
	cmd := exec.Command(javaPath, "-version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil
	}
	s := string(out)
	re := regexp.MustCompile(`version\s+"([^"]+)"`)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return false, nil
	}
	verStr := m[1]
	parts := strings.Split(verStr, ".")
	var major int
	if len(parts) > 0 {
		if parts[0] == "1" && len(parts) > 1 {
			sec := parts[1]
			num := regexp.MustCompile(`\d+`).FindString(sec)
			if num == "" {
				return false, nil
			}
			major, _ = strconv.Atoi(num)
		} else {
			num := regexp.MustCompile(`\d+`).FindString(parts[0])
			if num == "" {
				return false, nil
			}
			major, _ = strconv.Atoi(num)
		}
	}

	return major >= required, nil
}

func isTrueEnv(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes"
}
