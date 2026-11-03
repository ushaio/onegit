//go:build windows

package main

import (
	"os/exec"
	"testing"
)

// Every probe runs headless: a console window must not appear over the app.
func TestHideConsole(t *testing.T) {
	cmd := exec.Command("git", "--version")
	hideConsole(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("process attributes missing")
	}
	if !cmd.SysProcAttr.HideWindow || cmd.SysProcAttr.CreationFlags&0x08000000 == 0 {
		t.Fatalf("console not hidden: %+v", *cmd.SysProcAttr)
	}
}
