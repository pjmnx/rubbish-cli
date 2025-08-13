package status

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"rubbish/config"
	"rubbish/journal"
)

// helper to build a config with a fresh temporary journal
func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	j := &journal.Journal{Path: filepath.Join(dir, ".journal")}
	if err := j.Load(); err != nil {
		t.Fatalf("failed to load journal: %v", err)
	}
	cfg := &config.Config{
		WipeoutTime:   30,
		ContainerPath: dir,
		Journal:       j,
		WorkingDir:    filepath.Join(dir, "work"),
	}
	// create working dir
	os.MkdirAll(cfg.WorkingDir, 0o755)
	return cfg
}

// captureStdout runs fn while capturing stdout, returning printed text
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func md(item, origin string, wipeDays int, tossedAgo time.Duration) *journal.MetaData {
	return &journal.MetaData{
		Item:        item,
		Origin:      origin,
		WipeoutTime: wipeDays,
		TossedTime:  time.Now().Add(-tossedAgo).Unix(),
	}
}

func TestCommand_NoRecords(t *testing.T) {
	cfg := newTestConfig(t)
	globalLookup = false
	out := captureStdout(t, func() {
		if err := Command(nil, cfg); err != nil {
			t.Fatalf("Command returned error: %v", err)
		}
	})
	if !strings.Contains(out, "No rubbish found.") {
		t.Errorf("expected 'No rubbish found.' message, got: %s", out)
	}
}

func TestCommand_LocalRecords(t *testing.T) {
	cfg := newTestConfig(t)
	globalLookup = false

	// two records in working dir (one wipeable)
	r1 := md("old.txt", filepath.Join(cfg.WorkingDir, "old.txt"), 1, 48*time.Hour)
	r2 := md("new.txt", filepath.Join(cfg.WorkingDir, "sub/new.txt"), 10, 2*time.Hour)
	if err := cfg.Journal.AddRecord(r1); err != nil {
		t.Fatalf("add r1: %v", err)
	}
	if err := cfg.Journal.AddRecord(r2); err != nil {
		t.Fatalf("add r2: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Command(nil, cfg); err != nil {
			t.Fatalf("Command error: %v", err)
		}
	})

	// Output should include both records; implementation may not print a static header line
	if !strings.Contains(out, "old.txt") {
		t.Errorf("missing old.txt entry: %s", out)
	}
	// relative path should show sub/new.txt (with sub/ prefix)
	if !strings.Contains(out, "sub/new.txt") {
		t.Errorf("expected relative path for new.txt, got: %s", out)
	}
	if !strings.Contains(out, "Total: 2 | Wipable: 1") {
		t.Errorf("expected totals line, got: %s", out)
	}
}

func TestCommand_GlobalRecords(t *testing.T) {
	cfg := newTestConfig(t)
	globalLookup = true
	defer func() { globalLookup = false }()

	// record inside working dir
	r1 := md("inside.txt", filepath.Join(cfg.WorkingDir, "inside.txt"), 5, 24*time.Hour)
	// record outside working dir
	outsideDir := filepath.Join(cfg.ContainerPath, "other")
	os.MkdirAll(outsideDir, 0o755)
	r2 := md("outside.txt", filepath.Join(outsideDir, "outside.txt"), 1, 72*time.Hour)
	if err := cfg.Journal.AddRecord(r1); err != nil {
		t.Fatalf("add r1: %v", err)
	}
	if err := cfg.Journal.AddRecord(r2); err != nil {
		t.Fatalf("add r2: %v", err)
	}

	out := captureStdout(t, func() { _ = Command(nil, cfg) })
	if !strings.Contains(out, "Showing global rubbish status") {
		t.Errorf("missing global header: %s", out)
	}
	if !strings.Contains(out, "Total: 2") {
		t.Errorf("expected total 2, got: %s", out)
	}
	// global should not modify item names (no path join prefix injected)
	if strings.Contains(out, string(filepath.Separator)+"inside.txt") {
		t.Errorf("unexpected path modification in global mode: %s", out)
	}
}

func TestCommand_JournalError(t *testing.T) {
	// Journal not loaded -> db nil, causing error
	cfg := &config.Config{WorkingDir: "/tmp/doesnotmatter", Journal: &journal.Journal{}}
	globalLookup = false
	err := Command(nil, cfg)
	if err == nil || !strings.Contains(err.Error(), "journal database is not initialized") {
		t.Fatalf("expected journal initialization error, got: %v", err)
	}
}

func TestStringFormatting(t *testing.T) {
	// Remaining > 24h triggers day format
	m := md("file.txt", "/origin/file.txt", 10, 48*time.Hour) // 8 days remain
	s := String(m)
	if !strings.HasPrefix(s, "file.txt | Tossed:") {
		t.Errorf("unexpected prefix: %s", s)
	}
	if !strings.Contains(s, "WipeIn:8.0d") {
		t.Errorf("expected 8.0d remaining, got: %s", s)
	}

	// Remaining < 24h prints duration
	m2 := md("soon.txt", "/origin/soon.txt", 1, 12*time.Hour) // ~12h remain
	s2 := String(m2)
	re := regexp.MustCompile(`soon.txt \| Tossed:.* \| WipeIn:[0-9hms\.]+`)
	if !re.MatchString(s2) {
		t.Errorf("expected duration style remaining, got: %s", s2)
	}
}
