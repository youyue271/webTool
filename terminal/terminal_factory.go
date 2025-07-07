package terminal

import (
	"runtime"
)

type TerminalCreator func() (*Terminal, error)

func NewSystemTerminal() (*Terminal, error) {
	if runtime.GOOS == "windows" {
		return NewWindowsTerminal()
	}
	return NewUnixTerminal()
}

func NewWindowsTerminal() (*Terminal, error) {
	shell := "powershell.exe"
	return NewTerminal(shell)
}

func NewUnixTerminal() (*Terminal, error) {
	return NewTerminal("/bin/bash")
}
