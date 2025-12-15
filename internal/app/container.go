package app

import (
	"mc-manager/internal/backup"
	"mc-manager/internal/jvm"
	"mc-manager/internal/runner"
	"mc-manager/internal/server"
	"mc-manager/internal/storage"
	"mc-manager/internal/ws"
)

type Container struct {
	Store         *storage.GormStore
	JvmManager    *jvm.Manager
	ServerManager *server.Manager
	HubManager    *ws.HubManager
	Supervisor    *runner.Supervisor
	BackupManager *backup.Manager
}
