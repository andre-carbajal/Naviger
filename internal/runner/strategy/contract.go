package strategy

import "os/exec"

type ServerRunner interface {
	BuildCommand(javaPath string, absServerDir string, ram int) (*exec.Cmd, error)
}
