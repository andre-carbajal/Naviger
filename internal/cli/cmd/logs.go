package cmd

import (
	"naviger/internal/cli/ui"
)

func RunLogs(id string) {
	ui.RunLogs(Client, id)
}
