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

const BaseURL = "http://localhost:8080"

func printHelp() {
	fmt.Println("Uso: mc-cli <comando> [argumentos]")
	fmt.Println("\nComandos disponibles:")
	fmt.Println("  create         Crear un nuevo servidor.")
	fmt.Println("                 --name: Nombre del servidor (obligatorio)")
	fmt.Println("                 --version: Versión de Minecraft (obligatorio)")
	fmt.Println("                 --type: Tipo de loader, ej: vanilla (obligatorio)")
	fmt.Println("                 --ram: RAM en MB (obligatorio)")
	fmt.Println("\n  list           Listar todos los servidores.")
	fmt.Println("  start <id>     Iniciar un servidor por su ID.")
	fmt.Println("  stop <id>      Detener un servidor por su ID.")
	fmt.Println("  backup <id> [nombre] Crear un backup de un servidor. El nombre es opcional.")
	fmt.Println("  logs <id>      Ver la consola de un servidor y enviar comandos.")
	fmt.Println("  config ports   Gestionar el rango de puertos.")
	fmt.Println("                 --start: Puerto inicial")
	fmt.Println("                 --end: Puerto final")
	fmt.Println("  help           Muestra este mensaje de ayuda.")
}

func main() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Error al obtener el directorio de configuración del usuario: %v", err)
	}
	configDir := filepath.Join(userConfigDir, "mc-manager")

	_, err = config.LoadConfig(configDir)
	if err != nil {
		log.Fatalf("Error al cargar la configuración: %v", err)
	}

	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	stopCmd := flag.NewFlagSet("stop", flag.ExitOnError)
	backupCmd := flag.NewFlagSet("backup", flag.ExitOnError)
	logsCmd := flag.NewFlagSet("logs", flag.ExitOnError)
	configPortsCmd := flag.NewFlagSet("ports", flag.ExitOnError)
	helpCmd := flag.NewFlagSet("help", flag.ExitOnError)

	createName := createCmd.String("name", "", "Nombre del servidor")
	createVer := createCmd.String("version", "", "Versión de Minecraft")
	createType := createCmd.String("type", "", "Tipo (vanilla)")
	createRam := createCmd.Int("ram", 0, "RAM en MB")

	configPortStart := configPortsCmd.Int("start", 0, "Puerto inicial")
	configPortEnd := configPortsCmd.Int("end", 0, "Puerto final")

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		listCmd.Parse(os.Args[2:])
		handleList()

	case "create":
		createCmd.Parse(os.Args[2:])
		handleCreate(*createName, *createVer, *createType, *createRam)

	case "start":
		startCmd.Parse(os.Args[2:])
		if startCmd.NArg() < 1 {
			log.Fatal("Error: Debes especificar el ID del servidor. Ej: mc-cli start <UUID>")
		}
		handleStart(startCmd.Arg(0))

	case "stop":
		stopCmd.Parse(os.Args[2:])
		if stopCmd.NArg() < 1 {
			log.Fatal("Error: Debes especificar el ID del servidor.")
		}
		handleStop(stopCmd.Arg(0))

	case "backup":
		backupCmd.Parse(os.Args[2:])
		if backupCmd.NArg() < 1 {
			log.Fatal("Error: Debes especificar el ID del servidor. Ej: mc-cli backup <UUID> [nombre-opcional]")
		}
		serverID := backupCmd.Arg(0)
		backupName := ""
		if backupCmd.NArg() > 1 {
			backupName = backupCmd.Arg(1)
		}
		handleBackup(serverID, backupName)

	case "logs":
		logsCmd.Parse(os.Args[2:])
		if logsCmd.NArg() < 1 {
			log.Fatal("Error: Debes especificar el ID del servidor. Ej: mc-cli logs <UUID>")
		}
		handleLogs(logsCmd.Arg(0))

	case "config":
		if len(os.Args) < 3 {
			fmt.Println("Uso: mc-cli config [ports]")
			fmt.Println("\nSubcomandos:")
			fmt.Println("  ports          Ver o modificar rango de puertos")
			fmt.Println("                 Uso: mc-cli config ports [--start N --end N]")
			os.Exit(1)
		}

		switch os.Args[2] {
		case "ports":
			configPortsCmd.Parse(os.Args[3:])
			if *configPortStart == 0 && *configPortEnd == 0 {
				handleGetPortRange()
			} else {
				handleSetPortRange(*configPortStart, *configPortEnd)
			}
		default:
			fmt.Println("Subcomando desconocido:", os.Args[2])
			os.Exit(1)
		}

	case "help":
		helpCmd.Parse(os.Args[2:])
		printHelp()

	default:
		fmt.Println("Comando desconocido:", os.Args[1])
		printHelp()
		os.Exit(1)
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
					log.Printf("Error inesperado leyendo mensaje: %v", err)
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

func handleCreate(name, version, loaderType string, ram int) {
	if name == "" || version == "" || loaderType == "" || ram == 0 {
		log.Println("Error: Faltan argumentos para crear el servidor.")
		fmt.Println("\nUso correcto:")
		fmt.Println("  mc-cli create --name \"Mi Servidor\" --version \"1.20.1\" --type \"vanilla\" --ram 2048")
		os.Exit(1)
	}

	payload := map[string]interface{}{
		"name":    name,
		"version": version,
		"type":    loaderType,
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
	fmt.Println("Usa 'mc-cli list' para ver el estado.")
}

func handleStart(id string) {
	url := fmt.Sprintf("%s/servers/%s/start", BaseURL, id)
	resp, err := http.Post(url, "application/json", nil)
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
	url := fmt.Sprintf("%s/servers/%s/stop", BaseURL, id)
	resp, err := http.Post(url, "application/json", nil)
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
	url := fmt.Sprintf("%s/servers/%s/backup", BaseURL, id)

	payload := map[string]string{
		"name": name,
	}
	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
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
