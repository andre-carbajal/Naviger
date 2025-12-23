package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mc-manager/internal/config"
	"mc-manager/internal/domain"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/gorilla/websocket"
)

var BaseURL string

func printHelp() {
	prog := filepath.Base(os.Args[0])
	fmt.Printf("Uso: %s <recurso> <acción> [flags]\n\n", prog)
	fmt.Println("Recursos y acciones:")
	fmt.Printf("  %-60s %s\n", "server create --name <nombre> --version <versión> --loader <loader> --ram <MB>", "Crear nuevo servidor")
	fmt.Printf("  %-60s %s\n", "server list", "Listar servidores")
	fmt.Printf("  %-60s %s\n", "server start <id>", "Iniciar servidor")
	fmt.Printf("  %-60s %s\n", "server stop <id>", "Detener servidor")
	fmt.Printf("  %-60s %s\n", "server delete <id>", "Eliminar servidor")
	fmt.Printf("  %-60s %s\n", "server logs <id>", "Ver consola del servidor y enviar comandos")
	fmt.Println()
	fmt.Printf("  %-60s %s\n", "backup create <id> [nombre]", "Crear backup de servidor")
	fmt.Printf("  %-60s %s\n", "backup list [id]", "Listar backups (todos o por servidor)")
	fmt.Printf("  %-60s %s\n", "backup delete <nombre>", "Eliminar backup")
	fmt.Printf("  %-60s %s\n", "backup restore <nombre> --target <id>", "Restaurar backup en servidor existente")
	fmt.Printf("  %-60s %s\n", "backup restore <nombre> --new --name <nombre> --version <ver> --loader <loader> --ram <MB>", "Restaurar backup en servidor nuevo")
	fmt.Println()
	fmt.Printf("  %-60s %s\n", "ports get", "Mostrar rango de puertos")
	fmt.Printf("  %-60s %s\n", "ports set --start <n> --end <m>", "Establecer rango de puertos")
	fmt.Println()
	fmt.Printf("  %-60s %s\n", "loaders", "Muestra los loaders de servidores disponibles")
	fmt.Printf("  %-60s %s\n", "help", "Muestra este mensaje de ayuda")
	fmt.Println()
	fmt.Println("Ejemplo:")
	fmt.Printf("  %s server create --name \"Mi Servidor\" --version \"1.20.1\" --loader \"vanilla\" --ram 2048\n", prog)
}

func parseFlags(fs *flag.FlagSet, args []string, ctx string) {
	if err := fs.Parse(args); err != nil {
		log.Fatalf("Error parseando flags para %s: %v", ctx, err)
	}
}

func main() {
	flag.Usage = printHelp
	flag.Parse()

	port := config.GetPort()
	BaseURL = fmt.Sprintf("http://localhost:%d", port)

	args := flag.Args()
	if len(args) < 1 {
		printHelp()
		os.Exit(1)
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Error al obtener el directorio de configuración del usuario: %v", err)
	}
	configDir := filepath.Join(userConfigDir, "mc-manager")

	_, err = config.LoadConfig(configDir)
	if err != nil {
		log.Fatalf("Error al cargar la configuración: %v", err)
	}

	serverCreateCmd := flag.NewFlagSet("create", flag.ExitOnError)
	serverListCmd := flag.NewFlagSet("list", flag.ExitOnError)
	serverStartCmd := flag.NewFlagSet("start", flag.ExitOnError)
	serverStopCmd := flag.NewFlagSet("stop", flag.ExitOnError)
	serverDeleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)

	serverCreateName := serverCreateCmd.String("name", "", "Nombre del servidor")
	serverCreateVer := serverCreateCmd.String("version", "", "Versión de Minecraft")
	serverCreateLoader := serverCreateCmd.String("loader", "", "Loader (vanilla, paper, etc.)")
	serverCreateRam := serverCreateCmd.Int("ram", 0, "RAM en MB")

	backupCreateCmd := flag.NewFlagSet("create", flag.ExitOnError)
	backupListCmd := flag.NewFlagSet("list", flag.ExitOnError)
	backupDeleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	backupRestoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)

	backupRestoreTarget := backupRestoreCmd.String("target", "", "ID del servidor destino (para restaurar en existente)")
	backupRestoreNew := backupRestoreCmd.Bool("new", false, "Crear nuevo servidor desde backup")
	backupRestoreName := backupRestoreCmd.String("name", "", "Nombre del nuevo servidor")
	backupRestoreVer := backupRestoreCmd.String("version", "1.20.1", "Versión del nuevo servidor")
	backupRestoreLoader := backupRestoreCmd.String("loader", "vanilla", "Loader del nuevo servidor")
	backupRestoreRam := backupRestoreCmd.Int("ram", 2048, "RAM del nuevo servidor")

	portsGetCmd := flag.NewFlagSet("get", flag.ExitOnError)
	portsSetCmd := flag.NewFlagSet("set", flag.ExitOnError)
	portsSetStart := portsSetCmd.Int("start", 0, "Puerto inicial")
	portsSetEnd := portsSetCmd.Int("end", 0, "Puerto final")

	loadersCmd := flag.NewFlagSet("loaders", flag.ExitOnError)
	logsCmd := flag.NewFlagSet("logs", flag.ExitOnError)
	helpCmd := flag.NewFlagSet("help", flag.ExitOnError)

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "server":
		if len(cmdArgs) < 1 {
			fmt.Println("Uso: mc-cli server <subcomando>")
			fmt.Println("Subcomandos: create, list, start, stop, delete, logs")
			os.Exit(1)
		}
		sub := cmdArgs[0]
		subArgs := cmdArgs[1:]

		switch sub {
		case "create":
			parseFlags(serverCreateCmd, subArgs, "server create")
			handleCreate(*serverCreateName, *serverCreateVer, *serverCreateLoader, *serverCreateRam)

		case "list":
			parseFlags(serverListCmd, subArgs, "server list")
			handleList()

		case "start":
			parseFlags(serverStartCmd, subArgs, "server start")
			if serverStartCmd.NArg() < 1 {
				log.Fatal("Error: Debes especificar el ID del servidor. Ej: mc-cli server start <UUID>")
			}
			handleStart(serverStartCmd.Arg(0))

		case "stop":
			parseFlags(serverStopCmd, subArgs, "server stop")
			if serverStopCmd.NArg() < 1 {
				log.Fatal("Error: Debes especificar el ID del servidor.")
			}
			handleStop(serverStopCmd.Arg(0))

		case "delete":
			parseFlags(serverDeleteCmd, subArgs, "server delete")
			if serverDeleteCmd.NArg() < 1 {
				log.Fatal("Error: Debes especificar el ID del servidor.")
			}
			handleDelete(serverDeleteCmd.Arg(0))

		case "logs":
			parseFlags(logsCmd, subArgs, "server logs")
			if logsCmd.NArg() < 1 {
				log.Fatal("Error: Debes especificar el ID del servidor. Ej: mc-cli server logs <UUID>")
			}
			handleLogs(logsCmd.Arg(0))

		default:
			fmt.Println("Subcomando desconocido para 'server':", sub)
			os.Exit(1)
		}

	case "backup":
		if len(cmdArgs) < 1 {
			fmt.Println("Uso: mc-cli backup <subcomando>")
			fmt.Println("Subcomandos: create, list, delete, restore")
			os.Exit(1)
		}
		sub := cmdArgs[0]
		subArgs := cmdArgs[1:]

		switch sub {
		case "create":
			parseFlags(backupCreateCmd, subArgs, "backup create")
			if backupCreateCmd.NArg() < 1 {
				log.Fatal("Error: Debes especificar el ID del servidor. Ej: mc-cli backup create <UUID> [nombre-opcional]")
			}
			serverID := backupCreateCmd.Arg(0)
			backupName := ""
			if backupCreateCmd.NArg() > 1 {
				backupName = backupCreateCmd.Arg(1)
			}
			handleBackup(serverID, backupName)

		case "list":
			parseFlags(backupListCmd, subArgs, "backup list")
			if backupListCmd.NArg() > 0 {
				handleListBackups(backupListCmd.Arg(0))
			} else {
				handleListAllBackups()
			}

		case "delete":
			parseFlags(backupDeleteCmd, subArgs, "backup delete")
			if backupDeleteCmd.NArg() < 1 {
				log.Fatal("Error: Debes especificar el nombre del backup.")
			}
			handleDeleteBackup(backupDeleteCmd.Arg(0))

		case "restore":
			parseFlags(backupRestoreCmd, subArgs, "backup restore")
			if backupRestoreCmd.NArg() < 1 {
				log.Fatal("Error: Debes especificar el nombre del backup.")
			}
			backupName := backupRestoreCmd.Arg(0)
			handleRestoreBackup(backupName, *backupRestoreTarget, *backupRestoreNew, *backupRestoreName, *backupRestoreVer, *backupRestoreLoader, *backupRestoreRam)

		default:
			fmt.Println("Subcomando desconocido para 'backup':", sub)
			fmt.Println("Uso: mc-cli backup <subcomando>")
			fmt.Println("Subcomandos: create, list, delete, restore")
			os.Exit(1)
		}

	case "ports":
		if len(cmdArgs) < 1 {
			fmt.Println("Uso: mc-cli ports <subcomando>")
			fmt.Println("Subcomandos: get, set")
			os.Exit(1)
		}
		sub := cmdArgs[0]
		subArgs := cmdArgs[1:]

		switch sub {
		case "get":
			parseFlags(portsGetCmd, subArgs, "ports get")
			handleGetPortRange()

		case "set":
			parseFlags(portsSetCmd, subArgs, "ports set")
			if *portsSetStart == 0 || *portsSetEnd == 0 {
				log.Fatal("Error: Debes especificar ambos flags --start y --end para actualizar el rango de puertos")
			}
			handleSetPortRange(*portsSetStart, *portsSetEnd)

		default:
			fmt.Println("Subcomando desconocido para 'ports':", sub)
			fmt.Println("Uso: mc-cli ports <subcomando>")
			fmt.Println("Subcomandos: get, set")
			os.Exit(1)
		}

	case "loaders":
		parseFlags(loadersCmd, cmdArgs, "loaders")
		handleListLoaders()

	case "help":
		parseFlags(helpCmd, cmdArgs, "help")
		printHelp()

	default:
		fmt.Println("Comando desconocido:", command)
		printHelp()
		os.Exit(1)
	}
}

func handleListLoaders() {
	resp, err := http.Get(BaseURL + "/loaders")
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v\n(¿Está corriendo el servidor en otra terminal?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Error del servidor: %s", resp.Status)
	}

	var loaders []string
	if err := json.NewDecoder(resp.Body).Decode(&loaders); err != nil {
		log.Fatalf("Error leyendo respuesta: %v", err)
	}

	fmt.Println("\n--- LOADERS DISPONIBLES ---")
	for _, l := range loaders {
		fmt.Printf("- %s\n", l)
	}
}

func handleDelete(id string) {
	reqURL := fmt.Sprintf("%s/servers/%s", BaseURL, id)
	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		log.Fatalf("Error creando petición: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error eliminando servidor: %s", string(body))
	}

	fmt.Println("Servidor eliminado exitosamente.")
}

func handleDeleteBackup(name string) {
	reqURL := fmt.Sprintf("%s/backups/%s", BaseURL, name)
	req, err := http.NewRequest(http.MethodDelete, reqURL, nil)
	if err != nil {
		log.Fatalf("Error creando petición: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error eliminando backup: %s", string(body))
	}

	fmt.Println("Backup eliminado exitosamente.")
}

func handleRestoreBackup(backupName, targetID string, isNew bool, newName, newVer, newLoader string, newRam int) {
	reqURL := fmt.Sprintf("%s/backups/%s/restore", BaseURL, backupName)

	payload := map[string]interface{}{}

	if isNew {
		if newName == "" {
			log.Fatal("Error: Debes especificar --name para el nuevo servidor")
		}
		payload["newServerName"] = newName
		payload["newServerVersion"] = newVer
		payload["newServerLoader"] = newLoader
		payload["newServerRam"] = newRam
	} else {
		if targetID == "" {
			log.Fatal("Error: Debes especificar --target <ID> o usar --new")
		}
		payload["targetServerId"] = targetID
	}

	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(reqURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error restaurando backup: %s", string(body))
	}

	fmt.Println("Backup restaurado exitosamente.")
}

func handleListAllBackups() {
	reqURL := fmt.Sprintf("%s/backups", BaseURL)
	resp, err := http.Get(reqURL)
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error del servidor: %s", string(body))
	}

	var backups []domain.BackupInfo
	if err := json.NewDecoder(resp.Body).Decode(&backups); err != nil {
		log.Fatalf("Error leyendo respuesta: %v", err)
	}

	fmt.Println("\n--- TODOS LOS BACKUPS ---")
	for _, b := range backups {
		fmt.Printf("- %s (%.2f MB)\n", b.Name, float64(b.Size)/1024/1024)
	}
}

func handleListBackups(id string) {
	reqURL := fmt.Sprintf("%s/servers/%s/backups", BaseURL, id)
	resp, err := http.Get(reqURL)
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error del servidor: %s", string(body))
	}

	var backups []domain.BackupInfo
	if err := json.NewDecoder(resp.Body).Decode(&backups); err != nil {
		log.Fatalf("Error leyendo respuesta: %v", err)
	}

	fmt.Printf("\n--- BACKUPS PARA SERVIDOR %s ---\n", id)
	for _, b := range backups {
		fmt.Printf("- %s (%.2f MB)\n", b.Name, float64(b.Size)/1024/1024)
	}
}

func handleLogs(id string) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u, err := url.Parse(BaseURL)
	if err != nil {
		log.Fatal("Error parseando URL base:", err)
	}
	u.Scheme = "ws"
	wsURL := fmt.Sprintf("%s/ws/servers/%s/console", u.String(), id)

	fmt.Printf("Conectando a la consola de %s...\n", id)
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("Error al conectar al WebSocket. ¿Está el servidor corriendo? Error: %v", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("Error inesperado leyendo mensaje: %v", err)
				}
				fmt.Println("\nDesconectado de la consola.")
				return
			}
			fmt.Println(string(message))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			err := c.WriteMessage(websocket.TextMessage, []byte(input+"\n"))
			if err != nil {
				return
			}
		}
	}()

	fmt.Println("Conectado. Escribe comandos y presiona Enter. Presiona Ctrl+C para salir.")

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("Interrupción recibida, cerrando conexión...")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Error enviando mensaje de cierre:", err)
			}
			return
		}
	}
}

func handleList() {
	resp, err := http.Get(BaseURL + "/servers")
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v\n(¿Está corriendo el servidor en otra terminal?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Error del servidor: %s", resp.Status)
	}

	var servers []domain.Server
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		log.Fatalf("Error leyendo respuesta: %v", err)
	}

	fmt.Println("\n--- SERVIDORES REMOTOS ---")
	for _, s := range servers {
		statusIcon := "🔴"
		if s.Status == "RUNNING" {
			statusIcon = "🟢"
		} else if s.Status == "STARTING" {
			statusIcon = "🟡"
		}

		fmt.Printf("%s [%s] %s (v%s)\n", statusIcon, s.ID, s.Name, s.Version)
		fmt.Printf("      Port: %d | RAM: %dMB | Loader: %s\n", s.Port, s.RAM, s.Loader)
	}
}

func handleCreate(name, version, loader string, ram int) {
	if name == "" || version == "" || loader == "" || ram == 0 {
		log.Println("Error: Faltan argumentos para crear el servidor.")
		fmt.Println("\nUso correcto:")
		fmt.Println("  mc-cli server create --name \"Mi Servidor\" --version \"1.20.1\" --loader \"vanilla\" --ram 2048")
		os.Exit(1)
	}

	payload := map[string]interface{}{
		"name":    name,
		"version": version,
		"loader":  loader,
		"ram":     ram,
	}
	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(BaseURL+"/servers", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error creando servidor: %s", string(body))
	}

	fmt.Println("Petición de creación recibida.")
	fmt.Println("El servidor se está instalando en segundo plano.")
	fmt.Println("Usa 'mc-cli server list' para ver el estado.")
}

func handleStart(id string) {
	reqURL := fmt.Sprintf("%s/servers/%s/start", BaseURL, id)
	resp, err := http.Post(reqURL, "application/json", nil)
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Falló el inicio: %s", string(body))
	}

	fmt.Println("Orden de inicio enviada. El servidor arrancará en segundo plano.")
}

func handleStop(id string) {
	reqURL := fmt.Sprintf("%s/servers/%s/stop", BaseURL, id)
	resp, err := http.Post(reqURL, "application/json", nil)
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Falló la detención: %s", string(body))
	}

	fmt.Println("Orden de parada enviada.")
}

func handleBackup(id, name string) {
	reqURL := fmt.Sprintf("%s/servers/%s/backup", BaseURL, id)

	payload := map[string]string{
		"name": name,
	}
	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(reqURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Falló la creación del backup: %s", string(body))
	}

	var backupResponse struct {
		Message string `json:"message"`
		Path    string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&backupResponse); err != nil {
		log.Fatalf("Error leyendo respuesta del backup: %v", err)
	}

	fmt.Println(backupResponse.Message)
	fmt.Printf("Ubicación: %s\n", backupResponse.Path)
}

func handleGetPortRange() {
	resp, err := http.Get(BaseURL + "/settings/port-range")
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error obteniendo configuración: %s", string(body))
	}

	var portRange struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&portRange); err != nil {
		log.Fatalf("Error leyendo respuesta: %v", err)
	}

	fmt.Println("\n--- CONFIGURACIÓN DE PUERTOS ---")
	fmt.Printf("Puerto inicial: %d\n", portRange.Start)
	fmt.Printf("Puerto final:   %d\n", portRange.End)
	fmt.Printf("Rango:          %d puertos disponibles\n", portRange.End-portRange.Start+1)
}

func handleSetPortRange(start, end int) {
	if start == 0 || end == 0 {
		log.Fatal("Error: Debes especificar ambos puertos (--start y --end)")
	}

	if start > end {
		log.Fatal("Error: El puerto inicial debe ser menor o igual al puerto final")
	}

	payload := map[string]int{
		"start": start,
		"end":   end,
	}
	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPut, BaseURL+"/settings/port-range", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error creando petición: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error conectando al Daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Error actualizando configuración: %s", string(body))
	}

	fmt.Println("Configuración de puertos actualizada exitosamente!")
	fmt.Printf("Nuevo rango: %d - %d\n", start, end)
}
