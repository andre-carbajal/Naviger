package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var portsCmd = &cobra.Command{
	Use:   "ports",
	Short: "Manage port range",
}

var portsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get port range",
	Run: func(cmd *cobra.Command, args []string) {
		handleGetPortRange()
	},
}

var portsStart, portsEnd int
var portsSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set port range",
	Run: func(cmd *cobra.Command, args []string) {
		if portsStart == 0 || portsEnd == 0 {
			log.Fatal("Error: You must specify both --start and --end flags to update the port range")
		}
		handleSetPortRange(portsStart, portsEnd)
	},
}

var loadersCmd = &cobra.Command{
	Use:   "loaders",
	Short: "List available loaders",
	Run: func(cmd *cobra.Command, args []string) {
		handleListLoaders()
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for updates",
	Run: func(cmd *cobra.Command, args []string) {
		handleCheckUpdates()
	},
}

func init() {
	portsSetCmd.Flags().IntVar(&portsStart, "start", 0, "Start port")
	portsSetCmd.Flags().IntVar(&portsEnd, "end", 0, "End port")
	portsCmd.AddCommand(portsGetCmd, portsSetCmd)

	RootCmd.AddCommand(portsCmd, loadersCmd, updateCmd)
}

func handleGetPortRange() {
	pr, err := Client.GetPortRange()
	if err != nil {
		log.Fatalf("Error getting port range: %v", err)
	}
	fmt.Println("\n--- PORT CONFIGURATION ---")
	fmt.Printf("Start port: %d\n", pr.Start)
	fmt.Printf("End port:   %d\n", pr.End)
	fmt.Printf("Range:      %d ports available\n", pr.End-pr.Start+1)
}

func handleSetPortRange(start, end int) {
	if err := Client.SetPortRange(start, end); err != nil {
		log.Fatalf("Error setting port range: %v", err)
	}
	fmt.Println("Port configuration updated successfully!")
	fmt.Printf("New range: %d - %d\n", start, end)
}

func handleListLoaders() {
	loaders, err := Client.ListLoaders()
	if err != nil {
		log.Fatalf("Error listing loaders: %v", err)
	}
	fmt.Println("\n--- AVAILABLE LOADERS ---")
	for _, l := range loaders {
		fmt.Printf("- %s\n", l)
	}
}

func handleCheckUpdates() {
	info, err := Client.CheckUpdates()
	if err != nil {
		log.Fatalf("Error checking updates: %v", err)
	}

	fmt.Println("\n--- UPDATE CHECK ---")
	fmt.Printf("Current version: %s\n", info.CurrentVersion)
	fmt.Printf("Latest version:  %s\n", info.LatestVersion)

	if info.UpdateAvailable {
		fmt.Println("\nUpdate available!")
		fmt.Printf("Download it here: %s\n", info.ReleaseURL)
	} else {
		fmt.Println("\nYou are up to date.")
	}
}
