package main

import (
	"os/exec"
	"syscall"
)

// hideConsole keeps git from opening a console window for every probe, which
// would flash a terminal over the app while it scans folders or lists
// worktrees.
func hideConsole(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// HideWindow alone can still let a console briefly appear. Prevent its
	// creation as well; these commands communicate through pipes, not a TTY.
	const createNoWindow = 0x08000000 // CREATE_NO_WINDOW
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}
