package utils

import (
	"os"
	"os/exec"
	"proman/config"
)

func OpenDiff(file1, file2 string, label1, label2 string, config *config.Config) error {
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
			"--label",
			label1,
			file2,
			"--label",
			label2,
		)
	default: // git
		cmd = exec.Command(
			"git",
			"diff",
			"--no-index",
			file1,
			file2,
		)
		// git is a cli tool, so we don't want to detach it from the current process
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
	}

	cmd.SysProcAttr = detachProcess()

	return cmd.Start()
}
