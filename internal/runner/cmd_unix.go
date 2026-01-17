//go:build !windows

package runner

import (
	"os/exec"
)

func prepareCommand(cmd *exec.Cmd) {
	// No-op on non-Windows systems
}
