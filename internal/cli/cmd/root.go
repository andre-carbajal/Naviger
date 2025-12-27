package cmd

import (
	"fmt"
	"naviger/pkg/sdk"
	"os"

	"github.com/spf13/cobra"
)

var (
	Client  *sdk.Client
	BaseURL string
)

var RootCmd = &cobra.Command{
	Use:   "naviger-cli",
	Short: "CLI for Naviger Server Manager",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		Client = sdk.NewClient(BaseURL)
	},
	Run: func(cmd *cobra.Command, args []string) {
		RunDashboard()
	},
}

func Execute() {
	RootCmd.PersistentFlags().StringVar(&BaseURL, "url", "http://localhost:23008", "URL of the Naviger Daemon")

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
