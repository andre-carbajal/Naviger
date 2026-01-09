package strategy

import "os/exec"

type ServerRunner interface {
	BuildCommand(javaPath string, serverDir string, ram int, customArgs string) (*exec.Cmd, error)
}
