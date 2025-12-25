package app

import (
	"naviger/internal/backup"
	"naviger/internal/jvm"
	"naviger/internal/runner"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/ws"
)

type Container struct {
	Store         *storage.GormStore
	JvmManager    *jvm.Manager
	ServerManager *server.Manager
	HubManager    *ws.HubManager
	Supervisor    *runner.Supervisor
	BackupManager *backup.Manager
}
