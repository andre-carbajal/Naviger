package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"naviger/internal/api"
	"naviger/internal/app"
	"naviger/internal/backup"
	"naviger/internal/config"
	"naviger/internal/jvm"
	"naviger/internal/runner"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/ws"

	"github.com/emersion/go-autostart"
	"github.com/getlantern/systray"
	"github.com/pkg/browser"
)

//go:embed icon.png
var iconPngData []byte

//go:embed icon.ico
var iconIcoData []byte

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
	if runtime.GOOS == "windows" {
		systray.SetIcon(iconIcoData)
	} else {
		systray.SetIcon(iconPngData)
	}

	mStatus := systray.AddMenuItem("Status: Running", "Current status")
	mStatus.Disable()
	systray.AddSeparator()
	mOpenUI := systray.AddMenuItem("Open Web UI", "Open dashboard")
	mRestart := systray.AddMenuItem("Restart Daemon", "Reload configuration and restart server")
	mStartLogin := systray.AddMenuItemCheckbox("Start at Login", "Run on startup", false)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Quit application")

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
		}
	} else {
		log.Printf("Error getting executable: %v", err)
		mStartLogin.Disable()
	}

	var ctx context.Context
	var cancel context.CancelFunc
	var wg sync.WaitGroup

	startService := func() {
		ctx, cancel = context.WithCancel(context.Background())
		wg.Add(1)
		go func() {
			defer wg.Done()
			startDaemonService(ctx)
		}()
		mStatus.SetTitle("Status: Running")
	}

	startService()

	go func() {
		for {
			select {
			case <-mOpenUI.ClickedCh:
				port := config.GetPort()
				_ = browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))

			case <-mRestart.ClickedCh:
				mStatus.SetTitle("Status: Restarting...")
				log.Println("Reiniciando servicio...")

				cancel()
				wg.Wait()

				startService()
				log.Println("Servicio reiniciado.")

			case <-mStartLogin.ClickedCh:
				if appAutoStart == nil {
					continue
				}
				if mStartLogin.Checked() {
					if err := appAutoStart.Disable(); err == nil {
						mStartLogin.Uncheck()
					}
				} else {
					if err := appAutoStart.Enable(); err == nil {
						mStartLogin.Check()
					}
				}

			case <-mQuit.ClickedCh:
				mStatus.SetTitle("Status: Stopping...")
				cancel()
				wg.Wait()
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	log.Println("Application exited.")
}

func runHeadless() {
	log.Println("Running in headless mode...")

	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go startDaemonService(ctx)

	<-sigs
	log.Println("Signal received, shutting down...")
	cancel()

	time.Sleep(1 * time.Second)
}

func startDaemonService(ctx context.Context) {
	fmt.Println("Starting Naviger Daemon...")

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("Error getting config dir: %v", err)
		return
	}

	appName := "naviger"
	if config.IsDev() {
		appName = "naviger-dev"
	}
	configDir := filepath.Join(userConfigDir, appName)

	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		log.Printf("Error loading config: %v", err)
		return
	}

	for _, path := range []string{cfg.ServersPath, cfg.BackupsPath, cfg.RuntimesPath} {
		_ = os.MkdirAll(path, 0755)
	}

	store, err := storage.NewGormStore(cfg.DatabasePath)
	if err != nil {
		log.Printf("Fatal DB Error: %v", err)
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
		log.Printf("Warning resetting states: %v", err)
	}

	apiServer := api.NewAPIServer(container)
	listenAddr := fmt.Sprintf(":%d", config.GetPort())

	httpServer := apiServer.CreateHTTPServer(listenAddr)

	go func() {
		log.Printf("API Listening on %s", listenAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP Server Error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("Shutting down HTTP server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP Shutdown error: %v", err)
	}

	log.Println("Daemon stopped cleanly.")
}

func pngToIco(pngData []byte) ([]byte, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(pngData))
	if err != nil {
		return nil, err
	}

	// ICO Header: Reserved(2) + Type(2) + Count(2)
	header := []byte{0, 0, 1, 0, 1, 0}

	// Image Entry: W(1) + H(1) + Colors(1) + Res(1) + Planes(2) + BPP(2) + Size(4) + Offset(4)
	entry := make([]byte, 16)

	w := cfg.Width
	if w >= 256 {
		w = 0
	}
	entry[0] = byte(w)

	h := cfg.Height
	if h >= 256 {
		h = 0
	}
	entry[1] = byte(h)

	// Colors = 0, Reserved = 0

	// Planes = 1
	entry[4] = 1

	// BPP = 32
	entry[6] = 32

	// Size
	binary.LittleEndian.PutUint32(entry[8:], uint32(len(pngData)))

	// Offset = 6 (header) + 16 (entry) = 22
	binary.LittleEndian.PutUint32(entry[12:], 22)

	var buf bytes.Buffer
	buf.Write(header)
	buf.Write(entry)
	buf.Write(pngData)

	return buf.Bytes(), nil
}
