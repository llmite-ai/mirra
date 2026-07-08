// Package attach rewrites local CLI configuration (Claude Code, Codex) so
// that new sessions of those tools send their API traffic through the mirra
// proxy, and restores the original configuration on shutdown.
//
// What was changed is journaled to a state file before the proxy starts
// serving, so a run that dies without cleaning up can be repaired by the next
// `mirra start --attach` or by `mirra detach`.
package attach

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	TargetClaude = "claude"
	TargetCodex  = "codex"
)

// ErrNotInstalled marks a target whose config directory does not exist.
var ErrNotInstalled = errors.New("tool not detected")

// toolState records what Attach changed for one tool so it can be undone.
type toolState struct {
	ConfigPath  string `json:"config_path"`
	CreatedFile bool   `json:"created_file,omitempty"`
	// Claude only: prior value of env.ANTHROPIC_BASE_URL, if it was set.
	HadPrevValue bool   `json:"had_prev_value,omitempty"`
	PrevValue    string `json:"prev_value,omitempty"`
	CreatedEnv   bool   `json:"created_env,omitempty"`
}

// state is persisted to disk while attached so a crashed run can be repaired.
type state struct {
	ProxyURL   string                `json:"proxy_url"`
	AttachedAt time.Time             `json:"attached_at"`
	Tools      map[string]*toolState `json:"tools"`
}

type Manager struct {
	proxyURL string
	targets  []string
	log      *slog.Logger

	claudeSettingsPath string
	codexConfigPath    string
	statePath          string

	attached bool
	pretty   bool
}

// ParseTargets validates a comma-separated --attach value.
func ParseTargets(s string) ([]string, error) {
	var targets []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "" {
			continue
		}
		switch t {
		case TargetClaude, TargetCodex:
			if !slices.Contains(targets, t) {
				targets = append(targets, t)
			}
		default:
			return nil, fmt.Errorf("unknown attach target %q (supported: claude, codex)", t)
		}
	}
	if len(targets) == 0 {
		return nil, errors.New("no attach targets specified")
	}
	return targets, nil
}

func NewManager(targets []string, proxyURL string, log *slog.Logger) (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	claudeDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if claudeDir == "" {
		claudeDir = filepath.Join(home, ".claude")
	}
	codexDir := os.Getenv("CODEX_HOME")
	if codexDir == "" {
		codexDir = filepath.Join(home, ".codex")
	}

	return &Manager{
		proxyURL:           proxyURL,
		targets:            targets,
		log:                log,
		claudeSettingsPath: filepath.Join(claudeDir, "settings.json"),
		codexConfigPath:    filepath.Join(codexDir, "config.toml"),
		statePath:          filepath.Join(home, ".mirra", "attach.json"),
	}, nil
}

// Attach points every target tool's config at the proxy. Leftover state from
// a previous run is restored first so repeated or crashed runs don't stack
// overrides. Individual targets fail soft: a tool that isn't installed or
// can't be modified is logged and skipped.
func (m *Manager) Attach() error {
	if prev, err := m.loadState(); err != nil {
		m.log.Warn("could not read attach state; continuing", "path", m.statePath, "error", err)
	} else if prev != nil {
		if proxyAlive(prev.ProxyURL) {
			m.announceWarn(fmt.Sprintf("another mirra instance is attached (%s); taking over", prev.ProxyURL))
		} else {
			m.announceNote(fmt.Sprintf("repairing attach state left by a previous run (%s)", prev.ProxyURL))
		}
		m.restore(prev)
	}

	st := &state{ProxyURL: m.proxyURL, AttachedAt: time.Now(), Tools: map[string]*toolState{}}
	for _, target := range m.targets {
		var ts *toolState
		var err error
		switch target {
		case TargetClaude:
			ts, err = attachClaude(m.claudeSettingsPath, m.proxyURL)
		case TargetCodex:
			ts, err = attachCodex(m.codexConfigPath, m.proxyURL+"/v1")
		}
		if errors.Is(err, ErrNotInstalled) {
			m.announceSkipped(target)
			continue
		}
		if err != nil {
			m.announceAttachFailed(target, err)
			continue
		}
		st.Tools[target] = ts
		m.announceAttached(target, ts.ConfigPath)
	}
	m.blankLine()

	if len(st.Tools) == 0 {
		return errors.New("no tools attached")
	}
	if err := m.saveState(st); err != nil {
		// Without the state file a crash could not be repaired, so undo.
		m.restore(st)
		return fmt.Errorf("save attach state: %w", err)
	}
	m.attached = true
	return nil
}

// Detach restores the configs this manager attached. It is a no-op when
// Attach never succeeded or when another instance has taken over the state.
func (m *Manager) Detach() error {
	if !m.attached {
		return nil
	}
	st, err := m.loadState()
	if err != nil {
		return fmt.Errorf("read attach state: %w", err)
	}
	if st == nil {
		return nil
	}
	if st.ProxyURL != m.proxyURL {
		m.announceWarn(fmt.Sprintf("attach state now belongs to another mirra instance (%s); leaving it in place", st.ProxyURL))
		return nil
	}
	m.blankLine()
	m.restore(st)
	m.blankLine()
	m.attached = false
	return os.Remove(m.statePath)
}

// Restore undoes whatever the on-disk attach state records, regardless of
// which process created it. It backs the `mirra detach` command.
func Restore(log *slog.Logger, pretty bool) error {
	m, err := NewManager([]string{TargetClaude, TargetCodex}, "", log)
	if err != nil {
		return err
	}
	m.pretty = pretty
	st, err := m.loadState()
	if err != nil {
		return fmt.Errorf("read attach state: %w", err)
	}
	if st == nil {
		m.announceNote(fmt.Sprintf("nothing attached (%s)", m.statePath))
		return nil
	}
	if proxyAlive(st.ProxyURL) {
		m.announceWarn(fmt.Sprintf("a mirra instance still appears to be running (%s); detaching anyway", st.ProxyURL))
	}
	m.blankLine()
	m.restore(st)
	m.blankLine()
	return os.Remove(m.statePath)
}

func (m *Manager) restore(st *state) {
	for tool, ts := range st.Tools {
		var err error
		switch tool {
		case TargetClaude:
			err = detachClaude(ts, st.ProxyURL)
		case TargetCodex:
			err = detachCodex(ts)
		}
		if err != nil {
			m.announceRestoreFailed(tool, err)
			continue
		}
		m.announceRestored(tool, ts.ConfigPath)
	}
}

func (m *Manager) loadState() (*state, error) {
	data, err := os.ReadFile(m.statePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var st state
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (m *Manager) saveState(st *state) error {
	if err := os.MkdirAll(filepath.Dir(m.statePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(m.statePath, data, 0o600)
}

// proxyAlive reports whether a mirra proxy is responding at proxyURL.
func proxyAlive(proxyURL string) bool {
	if proxyURL == "" {
		return false
	}
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(proxyURL + "/health")
	if err != nil {
		return false
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return resp.StatusCode == http.StatusOK
}

// writeFileAtomic writes via a temp file and rename so a crash mid-write
// cannot leave a half-written config behind.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".mirra-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName) // no-op once renamed
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// fileModeOr returns path's current permissions, or fallback if it does not
// exist yet.
func fileModeOr(path string, fallback os.FileMode) os.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm()
	}
	return fallback
}
