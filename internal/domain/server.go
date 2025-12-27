package domain

import "time"

type Server struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Loader    string    `json:"loader"`
	Port      int       `json:"port"`
	RAM       int       `json:"ram"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type BackupInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type ProgressEvent struct {
	ServerID string `json:"serverId"`
	Message  string `json:"message"`
	Progress int    `json:"progress"`
}

type ServerStats struct {
	CPU  float64 `json:"cpu"`
	RAM  uint64  `json:"ram"`
	Disk int64   `json:"disk"`
}
