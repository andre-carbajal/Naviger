package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"naviger/internal/cli/ui"
	"naviger/pkg/sdk"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage servers",
}

var createName, createVer, createLoader string
var createRam int

var serverCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new server",
	Run: func(cmd *cobra.Command, args []string) {
		if createName != "" && createVer != "" && createLoader != "" && createRam != 0 {
			handleCreate(createName, createVer, createLoader, createRam)
			return
		}

		req, ok := ui.RunCreateWizard(Client)
		if ok {
			handleCreate(req.Name, req.Version, req.Loader, req.Ram)
		}
	},
}

var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all servers",
	Run: func(cmd *cobra.Command, args []string) {
		handleList()
	},
}

var serverStartCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "Start a server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleStart(args[0])
	},
}

var serverStopCmd = &cobra.Command{
	Use:   "stop [id]",
	Short: "Stop a server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleStop(args[0])
	},
}

var serverDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleDelete(args[0])
	},
}

var serverLogsCmd = &cobra.Command{
	Use:   "logs [id]",
	Short: "View server logs and console",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		RunLogs(args[0])
	},
}

func init() {
	serverCreateCmd.Flags().StringVar(&createName, "name", "", "Server name")
	serverCreateCmd.Flags().StringVar(&createVer, "version", "", "Minecraft version")
	serverCreateCmd.Flags().StringVar(&createLoader, "loader", "", "Loader (vanilla, paper, etc.)")
	serverCreateCmd.Flags().IntVar(&createRam, "ram", 0, "RAM in MB")

	serverCmd.AddCommand(serverCreateCmd, serverListCmd, serverStartCmd, serverStopCmd, serverDeleteCmd, serverLogsCmd)
	RootCmd.AddCommand(serverCmd)
}

func handleCreate(name, version, loader string, ram int) {
	requestID := uuid.New().String()

	req := sdk.CreateServerRequest{
		Name:      name,
		Version:   version,
		Loader:    loader,
		Ram:       ram,
		RequestID: requestID,
	}

	wsURL, err := Client.GetWebSocketURL(fmt.Sprintf("/ws/progress/%s", requestID))
	if err != nil {
		log.Fatal("Error parsing base URL:", err)
	}

	done := make(chan struct{})

	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("Warning: Could not connect to progress WebSocket: %v", err)
		close(done)
	} else {
		defer func() {
			_ = c.Close()
		}()
		go func() {
			defer close(done)
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					return
				}
				var event sdk.ProgressEvent
				if err := json.Unmarshal(message, &event); err == nil {
					fmt.Printf("\r[Progress] %s", event.Message)
					if event.Progress == 100 {
						fmt.Println()
						return
					}
				}
			}
		}()
	}

	if err := Client.CreateServer(req); err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	fmt.Println("\nCreation request received. Waiting for completion...")

	if c != nil {
		<-done
	}
}

func handleList() {
	ui.RunServerList(Client)
}

func handleStart(id string) {
	if err := Client.StartServer(id); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	fmt.Println("Start command sent.")
}

func handleStop(id string) {
	if err := Client.StopServer(id); err != nil {
		log.Fatalf("Error stopping server: %v", err)
	}
	fmt.Println("Stop command sent.")
}

func handleDelete(id string) {
	if err := Client.DeleteServer(id); err != nil {
		log.Fatalf("Error deleting server: %v", err)
	}
	fmt.Println("Server deleted successfully.")
}
