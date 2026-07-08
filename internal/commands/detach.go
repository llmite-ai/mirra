package commands

import (
	"log/slog"
	"os"

	"github.com/jpoz/mirra/internal/attach"
)

// Detach restores claude/codex configs from the attach state file. It exists
// for the case where an attached `mirra start` died without cleaning up.
func Detach(args []string) error {
	// LOG_OUTPUT drives the command logger in main; anything but json/plain
	// renders as the human-facing pretty format.
	format := os.Getenv("LOG_OUTPUT")
	pretty := format != "json" && format != "plain"
	return attach.Restore(slog.Default(), pretty)
}
