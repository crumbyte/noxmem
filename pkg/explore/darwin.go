//go:build darwin

package explore

import (
	"fmt"
	"os/exec"
)

func Explore(path string) error {
	if len(path) == 0 {
		return nil
	}

	cmd := exec.CommandContext(context.Background(), "open", path)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting open: %w", err)
	}

	go func() {
		_ = cmd.Wait()
	}()

	return nil
}
