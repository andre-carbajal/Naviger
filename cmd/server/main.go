package main

import (
	"fmt"
	"log"
	"mc-manager/internal/api"
	"mc-manager/internal/app"
	"mc-manager/internal/backup"
	"mc-manager/internal/config"
	"mc-manager/internal/jvm"
	"mc-manager/internal/runner"
	"mc-manager/internal/server"
	"mc-manager/internal/storage"
	"mc-manager/internal/ws"
	"os"
	"path/filepath"
)

func main() {
	fmt.Println("Iniciando Minecraft Manager Daemon...")

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Error al obtener el directorio de configuración del usuario: %v", err)
	}
	configDir := filepath.Join(userConfigDir, "mc-manager")

	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		log.Fatalf("Error al cargar la configuración: %v", err)
	}

	fmt.Printf("Usando base de datos: %s\n", cfg.DatabasePath)
	fmt.Printf("Usando directorio de servidores: %s\n", cfg.ServersPath)
	fmt.Printf("Usando directorio de runtimes de Java: %s\n", cfg.RuntimesPath)
	fmt.Printf("Usando directorio de backups: %s\n", cfg.BackupsPath)

	for _, path := range []string{cfg.ServersPath, cfg.BackupsPath, cfg.RuntimesPath} {
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatalf("Fatal: No se pudo crear el directorio '%s': %v", path, err)
		}
	}

	store, err := storage.NewGormStore(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Fatal: No se pudo conectar a la DB: %v", err)
	}

	jvmMgr := jvm.NewManager(cfg.RuntimesPath)
	srvMgr := server.NewManager(cfg.ServersPath, store)
	hubManager := ws.NewHubManager()
	supervisor := runner.NewSupervisor(store, jvmMgr, hubManager, cfg.ServersPath)
	backupManager := backup.NewManager(cfg.ServersPath, cfg.BackupsPath, store)

	container := &app.Container{
		Store:         store,
		JvmManager:    jvmMgr,
		ServerManager: srvMgr,
		HubManager:    hubManager,
		Supervisor:    supervisor,
		BackupManager: backupManager,
	}

	apiServer := api.NewAPIServer(container)

	listenAddr := fmt.Sprintf(":%d", config.GetPort())
	fmt.Printf("API Server escuchando en %s\n", listenAddr)

	if err := apiServer.Start(listenAddr); err != nil {
		log.Fatalf("Error API: %v", err)
	}
}
