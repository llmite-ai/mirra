package commands

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/jpoz/mirra/internal/config"
)

// Open launches the default browser at the mirra UI. It refuses to open a
// dead tab: if no mirra proxy answers on the configured port, it errors with
// a hint to start one instead.
func Open(args []string) error {
	fs := flag.NewFlagSet("open", flag.ExitOnError)
	port := fs.Int("port", 0, "Port the proxy is listening on")
	configPath := fs.String("config", "", "Path to config file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if *port != 0 {
		cfg.Port = *port
	}

	url := fmt.Sprintf("http://localhost:%d", cfg.Port)
	if !mirraAlive(url) {
		return fmt.Errorf("no mirra proxy answering at %s — start one with `mirra start`", url)
	}

	format := os.Getenv("LOG_OUTPUT")
	pretty := format != "json" && format != "plain"
	announce(pretty, fmt.Sprintf("opening %s", url))
	return openBrowser(url)
}

// openBrowser opens url in the system default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
