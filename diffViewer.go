package main

import (
	"os"
	"os/exec"
	"proman/config"
)

func openDiff(file1, file2 string, config *config.Config) {
	var cmd *exec.Cmd = nil
	switch config.Editor.Default {
	case "zed":
		cmd = exec.Command(
			"zed",
			"--diff",
			file1,
			file2,
		)
	case "vscode":
		cmd = exec.Command(
			"code",
			"--diff",
			file1,
			file2,
		)
	case "meld":
		cmd = exec.Command(
			"meld",
			file1,
			file2,
		)
	default: // git
		cmd = exec.Command(
			"git",
			"diff",
			"--no-index",
			file1,
			file2,
		)
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	cmd.Run()
}
