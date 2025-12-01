package main

import (
	"log"
	"mc-manager/internal/api"
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
	log.Println("Iniciando Minecraft Manager Daemon...")

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Error al obtener el directorio de configuración del usuario: %v", err)
	}
	configDir := filepath.Join(userConfigDir, "mc-manager")

	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		log.Fatalf("Error al cargar la configuración: %v", err)
	}

	log.Printf("Usando base de datos: %s", cfg.DatabasePath)
	log.Printf("Usando directorio de servidores: %s", cfg.ServersPath)
	log.Printf("Usando directorio de runtimes de Java: %s", cfg.RuntimesPath)
	log.Printf("Usando directorio de backups: %s", cfg.BackupsPath)

	for _, path := range []string{cfg.ServersPath, cfg.BackupsPath, cfg.RuntimesPath} {
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatalf("Fatal: No se pudo crear el directorio '%s': %v", path, err)
		}
	}

	store, err := storage.NewSQLiteStore(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Fatal: No se pudo conectar a la DB: %v", err)
	}

	jvmMgr := jvm.NewManager(cfg.RuntimesPath)

	srvMgr := server.NewManager(cfg.ServersPath, store)

	hubManager := ws.NewHubManager()

	supervisor := runner.NewSupervisor(store, jvmMgr, hubManager, cfg.ServersPath)

	backupManager := backup.NewManager(cfg.ServersPath, cfg.BackupsPath, store)

	apiServer := api.NewAPIServer(srvMgr, supervisor, store, hubManager, backupManager)

	if err := apiServer.Start("8080"); err != nil {
		log.Fatalf("Error API: %v", err)
	}
}
