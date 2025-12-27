package cmd

import (
	"naviger/internal/cli/ui"
)

func RunDashboard() {
	dashboardLoop := func() {
		for {
			serverID := ui.RunDashboard(Client)
			if serverID == "" {
				break
			}
			back := ui.RunLogs(Client, serverID)
			if !back {
				break
			}
		}
	}
	dashboardLoop()
}
