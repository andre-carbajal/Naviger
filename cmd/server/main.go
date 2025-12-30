package main

import (
	"fmt"
	"log"
	"naviger/internal/api"
	"naviger/internal/app"
	"naviger/internal/backup"
	"naviger/internal/config"
	"naviger/internal/jvm"
	"naviger/internal/runner"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/ws"
	"os"
	"path/filepath"
)

func main() {
	fmt.Println("Starting Naviger Daemon...")

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Error getting user config directory: %v", err)
	}
	appName := "naviger"
	if config.IsDev() {
		appName = "naviger-dev"
	}
	configDir := filepath.Join(userConfigDir, appName)

	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	fmt.Printf("Using database: %s\n", cfg.DatabasePath)
	fmt.Printf("Using servers directory: %s\n", cfg.ServersPath)
	fmt.Printf("Using Java runtimes directory: %s\n", cfg.RuntimesPath)
	fmt.Printf("Using backups directory: %s\n", cfg.BackupsPath)

	for _, path := range []string{cfg.ServersPath, cfg.BackupsPath, cfg.RuntimesPath} {
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatalf("Fatal: Could not create directory '%s': %v", path, err)
		}
	}

	store, err := storage.NewGormStore(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Fatal: Could not connect to DB: %v", err)
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

	if err := supervisor.ResetRunningStates(); err != nil {
		log.Printf("Warning: Failed to reset server states: %v", err)
	}

	apiServer := api.NewAPIServer(container)

	listenAddr := fmt.Sprintf(":%d", config.GetPort())
	fmt.Printf("API Server listening on %s\n", listenAddr)

	if err := apiServer.Start(listenAddr); err != nil {
		log.Fatalf("API Error: %v", err)
	}
}
