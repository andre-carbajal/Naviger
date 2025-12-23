package api

import (
	"encoding/json"
	"fmt"
	"mc-manager/internal/app"
	"mc-manager/internal/backup"
	"mc-manager/internal/domain"
	"mc-manager/internal/loader"
	"mc-manager/internal/runner"
	"mc-manager/internal/server"
	"mc-manager/internal/storage"
	"mc-manager/internal/ws"
	"net/http"
	"os"
	"path/filepath"
)

type Server struct {
	Manager       *server.Manager
	Supervisor    *runner.Supervisor
	Store         *storage.GormStore
	HubManager    *ws.HubManager
	BackupManager *backup.Manager
}

func NewAPIServer(container *app.Container) *Server {
	return &Server{
		Manager:       container.ServerManager,
		Supervisor:    container.Supervisor,
		Store:         container.Store,
		HubManager:    container.HubManager,
		BackupManager: container.BackupManager,
	}
}

func (api *Server) Start(listenAddr string) error {
	mux := http.NewServeMux()

	ex, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error obteniendo ruta del ejecutable: %v", err)
	}
	exPath := filepath.Dir(ex)
	webDistPath := filepath.Join(exPath, "web_dist")

	fs := http.FileServer(http.Dir(webDistPath))
	mux.Handle("/", fs)

	mux.HandleFunc("GET /loaders", api.handleGetLoaders)
	mux.HandleFunc("GET /loaders/{name}/versions", api.handleGetLoaderVersions)
	mux.HandleFunc("GET /servers", api.handleListServers)
	mux.HandleFunc("POST /servers", api.handleCreateServer)
	mux.HandleFunc("GET /servers/{id}", api.handleGetServer)
	mux.HandleFunc("PUT /servers/{id}", api.handleUpdateServer)
	mux.HandleFunc("DELETE /servers/{id}", api.handleDeleteServer)

	mux.HandleFunc("POST /servers/{id}/start", api.handleStartServer)
	mux.HandleFunc("POST /servers/{id}/stop", api.handleStopServer)
	mux.HandleFunc("POST /servers/{id}/backup", api.handleBackupServer)
	mux.HandleFunc("GET /servers/{id}/backups", api.handleListBackupsByServer)

	mux.HandleFunc("GET /backups", api.handleListAllBackups)
	mux.HandleFunc("DELETE /backups/{name}", api.handleDeleteBackup)
	mux.HandleFunc("POST /backups/{name}/restore", api.handleRestoreBackup)

	mux.HandleFunc("GET /settings/port-range", api.handleGetPortRange)
	mux.HandleFunc("PUT /settings/port-range", api.handleSetPortRange)

	mux.HandleFunc("GET /ws/servers/{id}/console", api.handleConsole)
	mux.HandleFunc("GET /ws/progress/{id}", api.handleProgress)

	handler := api.corsMiddleware(mux)

	fmt.Printf("API escuchando en http://0.0.0.0%s\n", listenAddr)
	return http.ListenAndServe(listenAddr, handler)
}

func (api *Server) handleGetLoaderVersions(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Falta el nombre del loader", http.StatusBadRequest)
		return
	}

	versions, err := loader.GetLoaderVersions(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

func (api *Server) handleGetLoaders(w http.ResponseWriter, r *http.Request) {
	loaders := loader.GetAvailableLoaders()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loaders)
}

func (api *Server) handleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Falta nombre del backup", http.StatusBadRequest)
		return
	}

	if err := api.BackupManager.DeleteBackup(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Falta nombre del backup", http.StatusBadRequest)
		return
	}

	var req struct {
		TargetServerID   string `json:"targetServerId"`
		NewServerName    string `json:"newServerName"`
		NewServerRAM     int    `json:"newServerRam"`
		NewServerLoader  string `json:"newServerLoader"`
		NewServerVersion string `json:"newServerVersion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if err := api.BackupManager.RestoreBackup(name, req.TargetServerID, req.NewServerName, req.NewServerRAM, req.NewServerLoader, req.NewServerVersion); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "restored"}`))
}

func (api *Server) handleListBackupsByServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Falta ID", http.StatusBadRequest)
		return
	}

	backups, err := api.BackupManager.ListBackups(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

func (api *Server) handleListAllBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := api.BackupManager.ListAllBackups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

func (api *Server) handleGetServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Falta ID", http.StatusBadRequest)
		return
	}

	srv, err := api.Manager.GetServer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if srv == nil {
		http.Error(w, "Servidor no encontrado", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(srv)
}

func (api *Server) handleUpdateServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Falta ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name *string `json:"name"`
		RAM  *int    `json:"ram"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if err := api.Store.UpdateServer(id, req.Name, req.RAM); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (api *Server) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Falta ID", http.StatusBadRequest)
		return
	}

	if err := api.Manager.DeleteServer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := api.Manager.ListServers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(servers)
}

func (api *Server) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		Version   string `json:"version"`
		Loader    string `json:"loader"`
		RAM       int    `json:"ram"`
		RequestID string `json:"requestId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	progressChan := make(chan string)
	hubID := "progress"
	if req.RequestID != "" {
		hubID = req.RequestID
	}
	hub := api.HubManager.GetHub(hubID)

	go func() {
		for msg := range progressChan {
			event := domain.ProgressEvent{
				ServerID: "new-server",
				Message:  msg,
				Progress: -1,
			}
			jsonBytes, _ := json.Marshal(event)
			hub.Broadcast(jsonBytes)
		}
	}()

	go func() {
		defer close(progressChan)
		srv, err := api.Manager.CreateServer(req.Name, req.Loader, req.Version, req.RAM, progressChan)
		if err != nil {
			fmt.Printf("Error creating server: %v\n", err)
			event := domain.ProgressEvent{
				ServerID: "error",
				Message:  fmt.Sprintf("Error: %v", err),
				Progress: 0,
			}
			jsonBytes, _ := json.Marshal(event)
			hub.Broadcast(jsonBytes)
			return
		}
		event := domain.ProgressEvent{
			ServerID: srv.ID,
			Message:  "Server created successfully",
			Progress: 100,
		}
		jsonBytes, _ := json.Marshal(event)
		hub.Broadcast(jsonBytes)

		// Clean up the hub after a short delay to allow message delivery
		// In a real app we might want a better cleanup strategy
		if req.RequestID != "" {
			// api.HubManager.RemoveHub(req.RequestID) // Maybe too aggressive if client hasn't received it yet
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "creating"}`))
}

func (api *Server) handleStartServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Falta ID", http.StatusBadRequest)
		return
	}

	if err := api.Supervisor.StartServer(id); err != nil {
		http.Error(w, fmt.Sprintf("Error iniciando: %v", err), http.StatusBadRequest)
		return
	}

	w.Write([]byte(`{"status": "started"}`))
}

func (api *Server) handleStopServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := api.Supervisor.StopServer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(`{"status": "stopping"}`))
}

func (api *Server) handleBackupServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Falta ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name string `json:"name,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	backupPath, err := api.BackupManager.CreateBackup(id, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "Backup creado exitosamente",
		"path":    backupPath,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *Server) handleGetPortRange(w http.ResponseWriter, r *http.Request) {
	start, end, err := api.Store.GetPortRange()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]int{
		"start": start,
		"end":   end,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *Server) handleSetPortRange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if err := api.Store.SetPortRange(req.Start, req.End); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "updated"}`))
}

func (api *Server) handleConsole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Falta ID", http.StatusBadRequest)
		return
	}

	hub := api.HubManager.GetHub(id)
	hub.ServeWs(w, r)
}

func (api *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Falta ID", http.StatusBadRequest)
		return
	}
	hub := api.HubManager.GetHub(id)
	hub.ServeWs(w, r)
}

func (api *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
