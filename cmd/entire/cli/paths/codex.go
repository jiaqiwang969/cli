package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

const codexHomeEnvVar = "CODEX_HOME"

// CodexHome returns the Codex home directory.
//
// It mirrors Codex's default behavior:
// - If CODEX_HOME is set, it is used.
// - Otherwise, defaults to ~/.codex.
func CodexHome() (string, error) {
	if value := os.Getenv(codexHomeEnvVar); value != "" {
		return value, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	return filepath.Join(homeDir, ".codex"), nil
}
