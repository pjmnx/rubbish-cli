package config_test

import (
	"os"
	"path/filepath"
	"rubbish/config"
	"testing"
)

// Helper to create a temporary INI file with given content
func createTempINI(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "config.ini")
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp INI: %v", err)
	}
	return file
}

func TestLoad_DefaultsAndOverrides(t *testing.T) {
	// System config (defaults)
	sysCfg := `[DEFAULT]
wipeout_time = 10
container_path = /tmp/rubbish
max_retention = 100
cleanup_interval = 2
[notifications]
enabled = true
days_in_advance = 3
timeout = 10
`
	// User config (overrides)
	userCfg := `[DEFAULT]
wipeout_time = 20
container_path = ~/custom_rubbish
[notifications]
enabled = false
days_in_advance = 5
`

	sysPath := createTempINI(t, sysCfg)
	userPath := createTempINI(t, userCfg)

	cfg, err := config.Load([]string{sysPath, userPath})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.WipeoutTime != 20 {
		t.Errorf("WipeoutTime override failed, got %d", cfg.WipeoutTime)
	}
	if cfg.ContainerPath == "/tmp/rubbish" {
		t.Errorf("ContainerPath should be expanded to user home, got %s", cfg.ContainerPath)
	}
	if cfg.Notification.Enabled != false {
		t.Errorf("Notification.Enabled override failed, got %v", cfg.Notification.Enabled)
	}
	if cfg.Notification.DaysInAdvance != 5 {
		t.Errorf("Notification.DaysInAdvance override failed, got %d", cfg.Notification.DaysInAdvance)
	}
	if cfg.MaxRetention != 100 {
		t.Errorf("MaxRetention should be 100, got %d", cfg.MaxRetention)
	}
	if cfg.CleanupInterval != 2 {
		t.Errorf("CleanupInterval should be 2, got %d", cfg.CleanupInterval)
	}
	if cfg.Notification.Timeout != 10 {
		t.Errorf("Notification.Timeout should be 10, got %d", cfg.Notification.Timeout)
	}
	if cfg.Journal == nil {
		t.Error("Journal should be initialized")
	}
	if cfg.WorkingDir == "" {
		t.Error("WorkingDir should be set")
	}
}

func TestLoad_ErrorCases(t *testing.T) {
	// Non-existent file should error
	if _, err := config.Load([]string{"/no/such/file.ini", "/no/such/user.ini"}); err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestLoad_InvalidIntegerValueGraceful(t *testing.T) {
	// Provide an invalid integer; library currently leaves default instead of erroring
	badSys := createTempINI(t, `wipeout_time = notANumber`)
	user := createTempINI(t, ``)
	cfg, err := config.Load([]string{badSys, user})
	if err != nil {
		t.Fatalf("did not expect error for invalid integer, got: %v", err)
	}
	if cfg.WipeoutTime != 30 { // default value from struct initialization
		t.Errorf("expected default wipeout_time 30 when invalid provided, got %d", cfg.WipeoutTime)
	}
}

func TestNormalizePath(t *testing.T) {
	// Absolute path
	abs := "/tmp/rubbish"
	if got := config.NormalizePath(abs); got != abs {
		t.Errorf("Absolute path not returned as-is: %s", got)
	}

	// Home expansion
	home, _ := os.UserHomeDir()
	rel := "~/rubbish"
	want := filepath.Join(home, "rubbish")
	if got := config.NormalizePath(rel); got != want {
		t.Errorf("Home path not expanded: got %s, want %s", got, want)
	}

	// Relative path
	rel2 := "rubbish"
	want2 := filepath.Join(home, "rubbish")
	if got := config.NormalizePath(rel2); got != want2 {
		t.Errorf("Relative path not expanded: got %s, want %s", got, want2)
	}
}

func TestJournalLoadFailure(t *testing.T) {
	// Simulate journal load failure by creating a directory with no write permission
	tmpDir := t.TempDir()
	journalDir := filepath.Join(tmpDir, "journal")
	if err := os.Mkdir(journalDir, 0555); err != nil {
		t.Fatalf("failed to create journal dir: %v", err)
	}
	sysCfg := createTempINI(t, "container_path = "+journalDir)
	userCfg := createTempINI(t, "")
	// Remove write permission so journal cannot be created
	os.Chmod(journalDir, 0555)
	_, err := config.Load([]string{sysCfg, userCfg})
	if err == nil {
		t.Error("Expected error when journal load fails")
	}
}
