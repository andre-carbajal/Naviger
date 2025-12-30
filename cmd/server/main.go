package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
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
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/emersion/go-autostart"
	"github.com/getlantern/systray"
	"github.com/pkg/browser"
)

//go:embed icon.png
var iconData []byte

var headless bool

func main() {
	flag.BoolVar(&headless, "headless", false, "Run in headless mode (no GUI)")
	flag.Parse()

	if headless {
		runHeadless()
	} else {
		runDesktop()
	}
}

func runDesktop() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTooltip("Naviger Daemon")

	if len(iconData) > 0 {
		systray.SetIcon(iconData)
	} else {
		log.Println("Warning: No icon data loaded")
	}

	mStatus := systray.AddMenuItem("Status: Running", "Current status")
	mStatus.Disable()

	systray.AddSeparator()

	mOpenUI := systray.AddMenuItem("Open Web UI", "Open dashboard in browser")
	mRestart := systray.AddMenuItem("Restart Daemon", "Restart the server")
	mStartLogin := systray.AddMenuItem("Start at Login", "Run on startup")

	executable, err := os.Executable()
	var appAutoStart *autostart.App
	if err == nil {
		appAutoStart = &autostart.App{
			Name:        "Naviger",
			DisplayName: "Naviger Daemon",
			Exec:        []string{executable},
		}

		if appAutoStart.IsEnabled() {
			mStartLogin.Check()
		} else {
			mStartLogin.Uncheck()
		}
	} else {
		log.Printf("Error getting executable path for autostart: %v", err)
		mStartLogin.Disable()
	}

	systray.AddSeparator()

	mExit := systray.AddMenuItem("Exit", "Quit the application")

	daemonStop := make(chan bool)
	go startDaemonService(daemonStop)

	go func() {
		for {
			select {
			case <-mOpenUI.ClickedCh:
				_ = browser.OpenURL("http://localhost:5173")
			case <-mRestart.ClickedCh:
				log.Println("Restarting daemon...")
				select {
				case daemonStop <- true:
				case <-time.After(1 * time.Second):
					log.Println("Timeout waiting for daemon to stop")
				}

				time.Sleep(500 * time.Millisecond)
				daemonStop = make(chan bool)
				go startDaemonService(daemonStop)
			case <-mStartLogin.ClickedCh:
				if appAutoStart == nil {
					continue
				}
				if mStartLogin.Checked() {
					if err := appAutoStart.Disable(); err != nil {
						log.Printf("Error disabling autostart: %v", err)
					} else {
						mStartLogin.Uncheck()
						log.Println("Start at Login disabled")
					}
				} else {
					if err := appAutoStart.Enable(); err != nil {
						log.Printf("Error enabling autostart: %v", err)
					} else {
						mStartLogin.Check()
						log.Println("Start at Login enabled")
					}
				}
			case <-mExit.ClickedCh:
				select {
				case daemonStop <- true:
				case <-time.After(1 * time.Second):
				}
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	log.Println("Exiting...")
}

func runHeadless() {
	stopChan := make(chan bool)
	go startDaemonService(stopChan)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	log.Println("Received signal, shutting down...")
	stopChan <- true
	time.Sleep(1 * time.Second)
}

func startDaemonService(stopChan chan bool) {
	fmt.Println("Starting Naviger Daemon...")

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("Error getting user config directory: %v", err)
		return
	}
	appName := "naviger"
	if config.IsDev() {
		appName = "naviger-dev"
	}
	configDir := filepath.Join(userConfigDir, appName)

	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		log.Printf("Error loading configuration: %v", err)
		return
	}

	fmt.Printf("Using database: %s\n", cfg.DatabasePath)
	fmt.Printf("Using servers directory: %s\n", cfg.ServersPath)
	fmt.Printf("Using Java runtimes directory: %s\n", cfg.RuntimesPath)
	fmt.Printf("Using backups directory: %s\n", cfg.BackupsPath)

	for _, path := range []string{cfg.ServersPath, cfg.BackupsPath, cfg.RuntimesPath} {
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Printf("Fatal: Could not create directory '%s': %v", path, err)
			return
		}
	}

	store, err := storage.NewGormStore(cfg.DatabasePath)
	if err != nil {
		log.Printf("Fatal: Could not connect to DB: %v", err)
		return
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

	httpServer := apiServer.CreateHTTPServer(listenAddr)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("API Error: %v", err)
		}
	}()

	<-stopChan
	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server exiting")
}
