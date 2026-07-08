package attach

import (
	"fmt"
	"os"
	"strings"
)

// ANSI styles for pretty output, matching the server startup banner. When
// pretty output is off (json/plain logging), events fall back to structured
// log records.
const (
	styleGood = "\033[1;92m" // bold bright green
	styleInfo = "\033[1;96m" // bold bright cyan
	styleWarn = "\033[1;93m" // bold bright yellow
	styleFail = "\033[1;91m" // bold bright red
	styleBold = "\033[1m"
	styleDim  = "\033[2m"
	styleURL  = "\033[4;96m" // underlined bright cyan, same as the banner URL
	styleOff  = "\033[0m"
)

// EnablePrettyOutput makes attach/detach events print as prominent banner
// lines on stdout instead of standard log records.
func (m *Manager) EnablePrettyOutput() {
	m.pretty = true
}

func (m *Manager) announceAttached(tool, configPath string) {
	if !m.pretty {
		m.log.Info("attached: new sessions will use the proxy", "tool", tool, "config", configPath)
		return
	}
	fmt.Fprintf(os.Stdout, "  %s⛓  ATTACHED%s  %s%s%s → %s%s%s\n",
		styleGood, styleOff, styleBold, strings.ToUpper(tool), styleOff, styleURL, m.proxyURL, styleOff)
	fmt.Fprintf(os.Stdout, "     %snew %s sessions will route through mirra · %s%s\n",
		styleDim, tool, shortenHome(configPath), styleOff)
}

func (m *Manager) announceRestored(tool, configPath string) {
	if !m.pretty {
		m.log.Info("restored", "tool", tool, "config", configPath)
		return
	}
	fmt.Fprintf(os.Stdout, "  %s⛓  DETACHED%s  %s%s%s · restored %s%s%s\n",
		styleInfo, styleOff, styleBold, strings.ToUpper(tool), styleOff, styleDim, shortenHome(configPath), styleOff)
}

func (m *Manager) announceAttachFailed(tool string, err error) {
	if !m.pretty {
		m.log.Error("attach failed", "tool", tool, "error", err)
		return
	}
	fmt.Fprintf(os.Stdout, "  %s✘  ATTACH FAILED%s  %s%s%s · %v\n",
		styleFail, styleOff, styleBold, strings.ToUpper(tool), styleOff, err)
}

func (m *Manager) announceRestoreFailed(tool string, err error) {
	if !m.pretty {
		m.log.Error("restore failed", "tool", tool, "error", err)
		return
	}
	fmt.Fprintf(os.Stdout, "  %s✘  RESTORE FAILED%s  %s%s%s · %v\n",
		styleFail, styleOff, styleBold, strings.ToUpper(tool), styleOff, err)
}

func (m *Manager) announceSkipped(tool string) {
	if !m.pretty {
		m.log.Info("attach skipped: tool not detected", "tool", tool)
		return
	}
	fmt.Fprintf(os.Stdout, "  %s−  %s not detected · skipped%s\n", styleDim, tool, styleOff)
}

func (m *Manager) announceWarn(msg string) {
	if !m.pretty {
		m.log.Warn(msg)
		return
	}
	fmt.Fprintf(os.Stdout, "  %s⚠  %s%s\n", styleWarn, msg, styleOff)
}

func (m *Manager) announceNote(msg string) {
	if !m.pretty {
		m.log.Info(msg)
		return
	}
	fmt.Fprintf(os.Stdout, "  %s◆  %s%s\n", styleDim, msg, styleOff)
}

// blankLine separates pretty event groups; a no-op for structured logging.
func (m *Manager) blankLine() {
	if m.pretty {
		fmt.Fprintln(os.Stdout)
	}
}

// shortenHome abbreviates the user's home directory to "~" for display.
func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if rest, ok := strings.CutPrefix(path, home+string(os.PathSeparator)); ok {
		return "~" + string(os.PathSeparator) + rest
	}
	return path
}
