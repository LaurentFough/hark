package clipboard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"hark/internal/sessionenv"
)

type Clipboard struct {
	Command string
}

func New() Clipboard {
	return Clipboard{Command: "wl-copy"}
}

func (c Clipboard) Copy(ctx context.Context, text string) error {
	if strings.TrimSpace(text) == "" {
		return errors.New("clipboard text must not be empty")
	}

	command := c.Command
	if command == "" {
		command = "wl-copy"
	}

	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf("%s is not installed or not in PATH", command)
	}

	// wl-copy forks a child that serves the selection and keeps stderr open, so
	// CombinedOutput would block until the clipboard changes. Wait on the
	// parent alone and read stderr only when it failed.
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	defer r.Close()
	cmd := sessionenv.Command(ctx, command)
	cmd.Stdin = strings.NewReader(text)
	cmd.Stdout, cmd.Stderr = w, w
	err = cmd.Start()
	w.Close()
	if err == nil {
		err = cmd.Wait()
	}
	if err == nil {
		return nil
	}
	_ = r.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	output, _ := io.ReadAll(io.LimitReader(r, 4096))
	message := strings.TrimSpace(string(output))
	if message == "" {
		message = err.Error()
	}
	return fmt.Errorf("%s failed: %s", command, message)
}
