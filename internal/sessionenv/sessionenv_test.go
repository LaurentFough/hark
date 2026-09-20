package sessionenv

import (
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func listen(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
}

func TestEnvironFillsUnsetVariables(t *testing.T) {
	dir := t.TempDir()
	listen(t, filepath.Join(dir, "wayland-1"))
	if err := os.WriteFile(filepath.Join(dir, "wayland-1.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	listen(t, filepath.Join(dir, "hypr", "sig-live", ".socket.sock"))
	if err := os.MkdirAll(filepath.Join(dir, "hypr", "sig-stale"), 0o700); err != nil {
		t.Fatal(err)
	}

	got := environ([]string{"HOME=/h"}, dir)
	for _, want := range []string{"HOME=/h", "WAYLAND_DISPLAY=wayland-1", "HYPRLAND_INSTANCE_SIGNATURE=sig-live"} {
		if !slices.Contains(got, want) {
			t.Errorf("environ missing %q: %v", want, got)
		}
	}
}

func TestEnvironPrefersNewestHyprlandInstance(t *testing.T) {
	dir := t.TempDir()
	listen(t, filepath.Join(dir, "hypr", "old", ".socket.sock"))
	listen(t, filepath.Join(dir, "hypr", "new", ".socket.sock"))
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "hypr", "old", ".socket.sock"), past, past); err != nil {
		t.Fatal(err)
	}

	if got := findHyprlandInstance(dir); got != "new" {
		t.Fatalf("findHyprlandInstance = %q, want new", got)
	}
}

func TestEnvironKeepsSetVariables(t *testing.T) {
	dir := t.TempDir()
	listen(t, filepath.Join(dir, "wayland-1"))
	listen(t, filepath.Join(dir, "hypr", "sig", ".socket.sock"))

	base := []string{"WAYLAND_DISPLAY=wayland-9", "HYPRLAND_INSTANCE_SIGNATURE="}
	got := environ(base, dir)
	if !slices.Equal(got, base) {
		t.Fatalf("environ changed set variables: %v", got)
	}
}

func TestEnvironWithoutSession(t *testing.T) {
	base := []string{"HOME=/h"}
	if got := environ(base, t.TempDir()); !slices.Equal(got, base) {
		t.Fatalf("environ = %v, want %v", got, base)
	}
}
