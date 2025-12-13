package main

import (
	"flag"
	"fmt"
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
	port := flag.Int("port", 0, "Puerto para ejecutar el servidor")
	flag.Parse()

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

	if *port != 0 {
		cfg.Port = *port
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

	listenAddr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("API Server escuchando en %s\n", listenAddr)

	if err := apiServer.Start(listenAddr); err != nil {
		log.Fatalf("Error API: %v", err)
	}
}
