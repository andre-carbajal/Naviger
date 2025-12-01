package api

import (
	"encoding/json"
	"fmt"
	"mc-manager/internal/backup"
	"mc-manager/internal/runner"
	"mc-manager/internal/server"
	"mc-manager/internal/storage"
	"mc-manager/internal/ws"
	"net/http"
)

type Server struct {
	Manager       *server.Manager
	Supervisor    *runner.Supervisor
	Store         *storage.SQLiteStore
	HubManager    *ws.HubManager
	BackupManager *backup.Manager
}

func NewAPIServer(mgr *server.Manager, sup *runner.Supervisor, store *storage.SQLiteStore, hubManager *ws.HubManager, backupManager *backup.Manager) *Server {
	return &Server{
		Manager:       mgr,
		Supervisor:    sup,
		Store:         store,
		HubManager:    hubManager,
		BackupManager: backupManager,
	}
}

func (api *Server) Start(port string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /servers", api.handleListServers)
	mux.HandleFunc("POST /servers", api.handleCreateServer)

	mux.HandleFunc("POST /servers/{id}/start", api.handleStartServer)
	mux.HandleFunc("POST /servers/{id}/stop", api.handleStopServer)
	mux.HandleFunc("POST /servers/{id}/backup", api.handleBackupServer)

	mux.HandleFunc("GET /settings/port-range", api.handleGetPortRange)
	mux.HandleFunc("PUT /settings/port-range", api.handleSetPortRange)

	mux.HandleFunc("GET /ws/servers/{id}/console", api.handleConsole)

	handler := api.corsMiddleware(mux)

	fmt.Printf("API escuchando en http://0.0.0.0:%s\n", port)
	return http.ListenAndServe(":"+port, handler)
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
		Name    string `json:"name"`
		Version string `json:"version"`
		Type    string `json:"type"`
		RAM     int    `json:"ram"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	srv, err := api.Manager.CreateServer(req.Name, req.Type, req.Version, req.RAM)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(srv)
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
