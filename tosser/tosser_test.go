package tosser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rubbish/config"
	"rubbish/journal"
)

func newTestCfg(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	j := &journal.Journal{Path: filepath.Join(dir, ".journal")}
	if err := j.Load(); err != nil {
		t.Fatalf("failed to init journal: %v", err)
	}
	return &config.Config{
		WipeoutTime:   1,
		ContainerPath: dir,
		Journal:       j,
		WorkingDir:    dir,
	}
}

func TestNameSufix_LengthAndCharset(t *testing.T) {
	s := NameSufix(12)
	if len(s) != 12 {
		t.Fatalf("expected length 12, got %d", len(s))
	}
	if strings.ToUpper(s) != s {
		t.Errorf("expected uppercase alphanumerics, got %s", s)
	}
	for _, ch := range s {
		if !(ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9') {
			t.Fatalf("unexpected character %q", ch)
		}
	}
}

func TestToss_FileMovedAndJournaled(t *testing.T) {
	cfg := newTestCfg(t)
	// create a file to toss
	src := filepath.Join(cfg.WorkingDir, "sample.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := Toss(src, cfg); err != nil {
		t.Fatalf("Toss returned error: %v", err)
	}

	// source should no longer exist
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected source removed, got err=%v", err)
	}

	// a file with randomized suffix should exist in container
	entries, _ := os.ReadDir(cfg.ContainerPath)
	var moved string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "sample.txt_") {
			moved = filepath.Join(cfg.ContainerPath, name)
			break
		}
	}
	if moved == "" {
		t.Fatalf("expected moved file with suffix in container")
	}

	// journal should contain record for the moved file
	base := filepath.Base(moved)
	md, err := cfg.Journal.Get(base)
	if err != nil || md == nil {
		t.Fatalf("journal record missing: %v", err)
	}
	if md.Item != base {
		t.Errorf("journal item mismatch: %s", md.Item)
	}
	if md.WipeoutTime != cfg.WipeoutTime {
		t.Errorf("wipeout time mismatch")
	}
}

func TestToss_RenameFailureCleansJournal(t *testing.T) {
	cfg := newTestCfg(t)
	// make containerPath unwritable so rename fails
	os.Chmod(cfg.ContainerPath, 0o400)
	defer os.Chmod(cfg.ContainerPath, 0o755)

	src := filepath.Join(cfg.WorkingDir, "bad.txt")
	os.WriteFile(src, []byte("x"), 0o644)
	err := Toss(src, cfg)
	if err == nil {
		t.Fatalf("expected error from Toss when rename fails")
	}
	// Ensure no lingering journal record for the attempted destination name
	entries, _ := os.ReadDir(cfg.ContainerPath)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "bad.txt_") {
			if _, jerr := cfg.Journal.Get(e.Name()); jerr == nil {
				t.Fatalf("unexpected journal record exists for failed toss: %s", e.Name())
			}
		}
	}
}

func TestCommand_SilentAndBinSize(t *testing.T) {
	cfg := newTestCfg(t)
	silentMode = true
	defer func() { silentMode = false }()
	// create small files
	for i := 0; i < 2; i++ {
		p := filepath.Join(cfg.WorkingDir, "f"+time.Now().Format("150405")+".txt")
		os.WriteFile(p, []byte("x"), 0o644)
		if err := Command([]string{p}, cfg); err != nil {
			t.Fatalf("command err: %v", err)
		}
	}
	// just ensure it returns nil and bin size is queryable
	if _, err := config.BinSize(cfg); err != nil {
		t.Fatalf("binsize error: %v", err)
	}
}

func TestCommand_InvalidArgs(t *testing.T) {
	cfg := newTestCfg(t)
	if err := Command([]string{}, cfg); err == nil {
		t.Fatalf("expected error for no args")
	}
	if err := Command([]string{"/this/does/not/exist"}, cfg); err == nil {
		t.Fatalf("expected error for invalid path")
	}
}
