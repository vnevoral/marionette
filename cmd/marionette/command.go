package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"marionette/internal/control"
)

// pairTimeout bounds how long `marionette pair` waits for the service.
const pairTimeout = 10 * time.Second

const usage = `Usage:
  marionette         run the service (configured by MARIONETTE_* environment variables)
  marionette pair    print a pairing code for a new device; run it on the host with sudo
  marionette help    show this help
`

// runCommand runs the command given on the command line (the service itself
// runs without arguments) and returns the process exit status.
func runCommand(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	switch args[0] {
	case "pair":
		if len(args) > 1 {
			_, _ = fmt.Fprintf(stderr, "marionette pair takes no arguments\n\n%s", usage)
			return 2
		}
		path, _ := controlSocketPath(getenv)
		return runPair(path, stdout, stderr)
	case "help", "-h", "--help":
		_, _ = io.WriteString(stdout, usage)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// runPair asks the running service for a pairing code over the control
// socket at path and prints it (FR-58).
func runPair(path string, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), pairTimeout)
	defer cancel()
	code, err := control.RequestCode(ctx, path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "marionette pair: %s\n", pairErrorMessage(err, path))
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "Pairing code: %s\nValid until:  %s (single use)\n\n"+
		"Open Marionette on the new device and enter the code on the pairing screen.\n"+
		"A new code replaces any code shown on the Devices page.\n",
		code.Code, code.ExpiresAt.Local().Format("2006-01-02 15:04:05 MST"))
	return 0
}

func pairErrorMessage(err error, path string) string {
	switch {
	case errors.Is(err, control.ErrNotRunning):
		return fmt.Sprintf("%v (%s); check `systemctl status marionette`", err, path)
	case errors.Is(err, control.ErrPermission):
		return fmt.Sprintf("%v (%s); run it with sudo", err, path)
	default:
		return err.Error()
	}
}
