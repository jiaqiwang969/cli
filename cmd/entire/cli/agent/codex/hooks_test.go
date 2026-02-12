package codex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallHooks_FreshInstall(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("ENTIRE_TEST_CODEX_CONFIG_PATH", "")

	ag := &CodexAgent{}
	count, err := ag.InstallHooks(false, false)
	if err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("InstallHooks() count = %d, want 1", count)
	}

	configPath := filepath.Join(tempDir, ".codex", "config.toml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("codex config not written: %v", err)
	}
	if !ag.AreHooksInstalled() {
		t.Fatal("AreHooksInstalled() = false, want true")
	}
}

func TestInstallHooks_Idempotent(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("ENTIRE_TEST_CODEX_CONFIG_PATH", "")

	ag := &CodexAgent{}
	if _, err := ag.InstallHooks(false, false); err != nil {
		t.Fatalf("first InstallHooks() error = %v", err)
	}

	count, err := ag.InstallHooks(false, false)
	if err != nil {
		t.Fatalf("second InstallHooks() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("second InstallHooks() count = %d, want 0", count)
	}
}

func TestInstallHooks_LocalDevSwitch(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("ENTIRE_TEST_CODEX_CONFIG_PATH", "")

	ag := &CodexAgent{}
	if _, err := ag.InstallHooks(false, false); err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	count, err := ag.InstallHooks(true, false)
	if err != nil {
		t.Fatalf("InstallHooks(localDev) error = %v", err)
	}
	if count != 1 {
		t.Fatalf("InstallHooks(localDev) count = %d, want 1", count)
	}

	config, err := readCodexConfig(codexConfigPath())
	if err != nil {
		t.Fatalf("readCodexConfig() error = %v", err)
	}
	notify := codexNotifyFromConfig(config)
	if len(notify) == 0 || notify[0] != "go" {
		t.Fatalf("local-dev notify command = %v, want go run ...", notify)
	}
}

func TestUninstallHooks(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("ENTIRE_TEST_CODEX_CONFIG_PATH", "")

	ag := &CodexAgent{}
	if _, err := ag.InstallHooks(false, false); err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}
	if err := ag.UninstallHooks(); err != nil {
		t.Fatalf("UninstallHooks() error = %v", err)
	}
	if ag.AreHooksInstalled() {
		t.Fatal("AreHooksInstalled() = true, want false")
	}
}

func TestUninstallHooks_PreservesForeignNotify(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("ENTIRE_TEST_CODEX_CONFIG_PATH", "")

	configPath := codexConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o750); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`notify = ["osascript", "-e", "beep"]`), 0o600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	ag := &CodexAgent{}
	if err := ag.UninstallHooks(); err != nil {
		t.Fatalf("UninstallHooks() error = %v", err)
	}

	config, err := readCodexConfig(configPath)
	if err != nil {
		t.Fatalf("readCodexConfig() error = %v", err)
	}
	notify := codexNotifyFromConfig(config)
	if len(notify) != 3 || notify[0] != "osascript" {
		t.Fatalf("foreign notify command unexpectedly changed: %v", notify)
	}
}
