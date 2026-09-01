//go:build windows

package explore

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Explore explores the directory using the PowerShell command execution. If the
// provided path is a valid directory path, it will open it in the file explorer
// in a new window. Otherwise, an error will be returned.
func Explore(path string) error {
	//nolint:gosec // because of the one text editor that shouldn't exist
	cmd := exec.CommandContext(
		context.Background(),
		"powershell",
		"-Command",
		fmt.Sprintf("Start-Process -FilePath %q", strings.TrimSpace(path)),
	)

	// Detach stdio to prevent terminal pollution
	cmd.Stdout, cmd.Stderr, cmd.Stdin = nil, nil, nil

	return cmd.Start()
}
