package info

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

// helpers
func newTestCfg(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	j := &journal.Journal{Path: filepath.Join(dir, ".journal")}
	if err := j.Load(); err != nil {
		t.Fatalf("failed to load journal: %v", err)
	}
	return &config.Config{ContainerPath: dir, Journal: j, WipeoutTime: 2}
}

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

func TestCommand_ByName_PrintsDetails(t *testing.T) {
	cfg := newTestCfg(t)
	// add one record
	rec := md("file.txt", "/path/to/file.txt", 3, 12*time.Hour)
	if err := cfg.Journal.AddRecord(rec); err != nil {
		t.Fatalf("add record: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Command([]string{"/any/dir/file.txt"}, cfg); err != nil {
			t.Fatalf("command error: %v", err)
		}
	})

	if !strings.Contains(out, "Item: file.txt") {
		t.Errorf("missing item line: %s", out)
	}
	if !strings.Contains(out, "Origin: /path/to/file.txt") {
		t.Errorf("missing origin line: %s", out)
	}
	// Wipeable At should be a date string
	re := regexp.MustCompile(`Wipeable At: \d{4}-\d{2}-\d{2}`)
	if !re.MatchString(out) {
		t.Errorf("missing wipeable date line: %s", out)
	}
	if !strings.Contains(out, "Remaining:") {
		t.Errorf("missing remaining line: %s", out)
	}
}

func TestCommand_MissingArg(t *testing.T) {
	cfg := newTestCfg(t)
	if err := Command([]string{}, cfg); err == nil || !strings.Contains(err.Error(), "item name is required") {
		t.Fatalf("expected missing arg error, got: %v", err)
	}
}

func TestCommand_ByName_NotFound(t *testing.T) {
	cfg := newTestCfg(t)
	err := Command([]string{"ghost.txt"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "failed to get item") {
		t.Fatalf("expected failed to get item error, got: %v", err)
	}
}

func TestCommand_ByPosition_Positive(t *testing.T) {
	cfg := newTestCfg(t)
	// Insert records with lexicographically ordered keys
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := cfg.Journal.AddRecord(md(name, "/o/"+name, 2, time.Hour)); err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
	}
	byPosition = 2
	defer func() { byPosition = 0 }()

	out := captureStdout(t, func() {
		if err := Command(nil, cfg); err != nil {
			t.Fatalf("command error: %v", err)
		}
	})
	if !strings.Contains(out, "Item: b.txt") {
		t.Errorf("expected item b.txt in output, got: %s", out)
	}
}

func TestCommand_ByPosition_NegativeSelectsFromEnd(t *testing.T) {
	cfg := newTestCfg(t)
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := cfg.Journal.AddRecord(md(name, "/o/"+name, 2, time.Hour)); err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
	}
	byPosition = -1
	defer func() { byPosition = 0 }()
	out := captureStdout(t, func() {
		if err := Command(nil, cfg); err != nil {
			t.Fatalf("command error: %v", err)
		}
	})
	if !strings.Contains(out, "Item: c.txt") {
		t.Errorf("expected last item c.txt, got: %s", out)
	}
}

func TestCommand_ByPosition_IndexOutOfRange(t *testing.T) {
	cfg := newTestCfg(t)
	if err := cfg.Journal.AddRecord(md("only.txt", "/o/only.txt", 2, time.Hour)); err != nil {
		t.Fatal(err)
	}
	byPosition = 5
	defer func() { byPosition = 0 }()
	err := Command(nil, cfg)
	if err == nil || !strings.Contains(err.Error(), "invalid item position: 5") {
		t.Fatalf("expected invalid position error, got: %v", err)
	}
}

func TestCommand_ByPosition_ListError(t *testing.T) {
	// Uninitialized journal causes list error
	cfg := &config.Config{Journal: &journal.Journal{}}
	byPosition = 1
	defer func() { byPosition = 0 }()
	err := Command(nil, cfg)
	if err == nil || !strings.Contains(err.Error(), "failed to list items") {
		t.Fatalf("expected list error, got: %v", err)
	}
}

func TestCommand_OverdueRemaining(t *testing.T) {
	cfg := newTestCfg(t)
	rec := md("old.txt", "/o/old.txt", 1, 72*time.Hour) // 3 days ago, wipe after 1 -> overdue
	if err := cfg.Journal.AddRecord(rec); err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() { _ = Command([]string{"old.txt"}, cfg) })
	if !strings.Contains(out, "(overdue)") {
		t.Fatalf("expected overdue annotation, got: %s", out)
	}
}
