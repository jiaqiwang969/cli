package codex

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/pelletier/go-toml/v2"
)

// Ensure CodexAgent implements HookSupport and HookHandler.
var (
	_ agent.HookSupport = (*CodexAgent)(nil)
	_ agent.HookHandler = (*CodexAgent)(nil)
)

var (
	codexNotifyCommand         = []string{"entire", "hooks", "codex", HookNameNotify}
	codexNotifyLocalDevCommand = []string{
		"go",
		"run",
		"${CODEX_PROJECT_DIR}/cmd/entire/main.go",
		"hooks",
		"codex",
		HookNameNotify,
	}
)

// GetHookNames returns the hook verbs Codex supports.
func (c *CodexAgent) GetHookNames() []string {
	return []string{HookNameNotify}
}

// InstallHooks installs Codex notify hook in ~/.codex/config.toml.
func (c *CodexAgent) InstallHooks(localDev bool, force bool) (int, error) {
	configPath := codexConfigPath()
	config, err := readCodexConfig(configPath)
	if err != nil {
		return 0, err
	}

	target := codexNotifyCommand
	if localDev {
		target = codexNotifyLocalDevCommand
	}

	current := codexNotifyFromConfig(config)
	if !force && slices.Equal(current, target) {
		return 0, nil
	}

	config["notify"] = target
	if err := writeCodexConfig(configPath, config); err != nil {
		return 0, err
	}
	return 1, nil
}

// UninstallHooks removes Codex notify hook only when it points to Entire.
func (c *CodexAgent) UninstallHooks() error {
	configPath := codexConfigPath()
	config, err := readCodexConfig(configPath)
	if err != nil {
		return err
	}

	current := codexNotifyFromConfig(config)
	if !isEntireCodexNotifyCommand(current) {
		return nil
	}

	delete(config, "notify")
	return writeCodexConfig(configPath, config)
}

// AreHooksInstalled checks if Codex notify is wired to Entire hooks.
func (c *CodexAgent) AreHooksInstalled() bool {
	config, err := readCodexConfig(codexConfigPath())
	if err != nil {
		return false
	}
	return isEntireCodexNotifyCommand(codexNotifyFromConfig(config))
}

// GetSupportedHooks returns the normalized hook types used by Codex notify.
func (c *CodexAgent) GetSupportedHooks() []agent.HookType {
	return []agent.HookType{
		agent.HookStop,
		agent.HookPostToolUse,
	}
}

func codexNotifyFromConfig(config map[string]any) []string {
	rawNotify, ok := config["notify"]
	if !ok {
		return nil
	}

	switch values := rawNotify.(type) {
	case []string:
		return values
	case []any:
		notify := make([]string, 0, len(values))
		for _, value := range values {
			str, ok := value.(string)
			if !ok {
				return nil
			}
			notify = append(notify, str)
		}
		return notify
	default:
		return nil
	}
}

func isEntireCodexNotifyCommand(argv []string) bool {
	if len(argv) < 4 {
		return false
	}
	n := len(argv)
	return argv[n-3] == "hooks" && argv[n-2] == "codex" && argv[n-1] == HookNameNotify
}

func readCodexConfig(configPath string) (map[string]any, error) {
	data, err := os.ReadFile(configPath) //nolint:gosec // path is from user home + fixed suffix
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("failed to read codex config: %w", err)
	}

	config := make(map[string]any)
	if len(bytes.TrimSpace(data)) == 0 {
		return config, nil
	}
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse codex config: %w", err)
	}
	return config, nil
}

func writeCodexConfig(configPath string, config map[string]any) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create codex config directory: %w", err)
	}

	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to encode codex config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write codex config: %w", err)
	}
	return nil
}
