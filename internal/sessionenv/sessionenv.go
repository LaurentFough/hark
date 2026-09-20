// Package sessionenv recovers Wayland and Hyprland session variables for
// subprocesses when the daemon started before the compositor exported them.
package sessionenv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	waylandDisplay = "WAYLAND_DISPLAY"
	hyprlandSig    = "HYPRLAND_INSTANCE_SIGNATURE"
)

// Command is exec.CommandContext with the recovered session environment.
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = Environ()
	return cmd
}

// Getenv reads key from the environment with session fallbacks applied.
func Getenv(key string) string {
	prefix := key + "="
	for _, entry := range Environ() {
		if value, ok := strings.CutPrefix(entry, prefix); ok {
			return value
		}
	}
	return ""
}

// Environ returns the process environment plus any session variable that is
// unset (an explicitly empty value is respected) and discoverable on disk.
func Environ() []string {
	return environ(os.Environ(), runtimeDir())
}

func environ(base []string, dir string) []string {
	set := make(map[string]bool, len(base))
	for _, entry := range base {
		if key, _, ok := strings.Cut(entry, "="); ok {
			set[key] = true
		}
	}
	out := base
	if !set[waylandDisplay] {
		if name := findWaylandSocket(dir); name != "" {
			out = append(out[:len(out):len(out)], waylandDisplay+"="+name)
		}
	}
	if !set[hyprlandSig] {
		if sig := findHyprlandInstance(dir); sig != "" {
			out = append(out[:len(out):len(out)], hyprlandSig+"="+sig)
		}
	}
	return out
}

func runtimeDir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return dir
	}
	return fmt.Sprintf("/run/user/%d", os.Getuid())
}

func findWaylandSocket(dir string) string {
	matches, _ := filepath.Glob(filepath.Join(dir, "wayland-*"))
	sort.Strings(matches)
	for _, path := range matches {
		if strings.HasSuffix(path, ".lock") {
			continue
		}
		if info, err := os.Stat(path); err == nil && info.Mode()&os.ModeSocket != 0 {
			return filepath.Base(path)
		}
	}
	return ""
}

// findHyprlandInstance picks the newest instance directory that holds a live
// command socket; older crashed sessions can leave stale directories behind.
func findHyprlandInstance(dir string) string {
	entries, _ := os.ReadDir(filepath.Join(dir, "hypr"))
	var (
		best     string
		bestTime int64
	)
	for _, entry := range entries {
		info, err := os.Stat(filepath.Join(dir, "hypr", entry.Name(), ".socket.sock"))
		if err != nil || info.Mode()&os.ModeSocket == 0 {
			continue
		}
		if t := info.ModTime().UnixNano(); best == "" || t > bestTime {
			best, bestTime = entry.Name(), t
		}
	}
	return best
}
