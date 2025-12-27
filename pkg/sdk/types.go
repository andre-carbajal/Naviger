package sdk

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

type UpdateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url"`
}

type PortRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type CreateServerRequest struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Loader    string `json:"loader"`
	Ram       int    `json:"ram"`
	RequestID string `json:"requestId"`
}

type RestoreBackupRequest struct {
	TargetServerID   string `json:"targetServerId,omitempty"`
	NewServerName    string `json:"newServerName,omitempty"`
	NewServerVersion string `json:"newServerVersion,omitempty"`
	NewServerLoader  string `json:"newServerLoader,omitempty"`
	NewServerRam     int    `json:"newServerRam,omitempty"`
}
