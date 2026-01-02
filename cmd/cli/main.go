package main

import (
	"naviger/internal/cli/cmd"
	"naviger/internal/config"
)

func main() {
	port := config.GetPort()
	cmd.Execute(port)
}
