package clipboard

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCopyRejectsEmptyText(t *testing.T) {
	clip := Clipboard{Command: "wl-copy"}
	if err := clip.Copy(context.Background(), " \n\t "); err == nil {
		t.Fatal("expected empty text error")
	}
}

func TestCopyReturnsCommandFailure(t *testing.T) {
	command := filepath.Join(t.TempDir(), "wl-copy")
	if err := os.WriteFile(command, []byte("#!/bin/sh\necho clipboard unavailable >&2\nexit 23\n"), 0o700); err != nil {
		t.Fatalf("write fake wl-copy: %v", err)
	}

	err := (Clipboard{Command: command}).Copy(context.Background(), "new clipboard text")
	if err == nil || !strings.Contains(err.Error(), "clipboard unavailable") {
		t.Fatalf("Copy error = %v, want command failure", err)
	}
}

func TestCopyReturnsWhileForkedChildKeepsStderr(t *testing.T) {
	command := filepath.Join(t.TempDir(), "wl-copy")
	script := "#!/bin/sh\ncat >/dev/null\nsleep 30 &\nexit 0\n"
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake wl-copy: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- (Clipboard{Command: command}).Copy(context.Background(), "text") }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Copy error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Copy blocked on a background child holding stderr")
	}
}
