package attach

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testProxyURL = "http://localhost:4567"

func testManager(t *testing.T, targets ...string) *Manager {
	t.Helper()
	root := t.TempDir()
	claudeDir := filepath.Join(root, "claude")
	codexDir := filepath.Join(root, "codex")
	require.NoError(t, os.MkdirAll(claudeDir, 0o755))
	require.NoError(t, os.MkdirAll(codexDir, 0o755))
	return &Manager{
		proxyURL:           testProxyURL,
		targets:            targets,
		log:                slog.Default(),
		claudeSettingsPath: filepath.Join(claudeDir, "settings.json"),
		codexConfigPath:    filepath.Join(codexDir, "config.toml"),
		statePath:          filepath.Join(root, "mirra", "attach.json"),
	}
}

func readClaudeSettings(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings))
	return settings
}

func claudeBaseURL(t *testing.T, path string) (string, bool) {
	t.Helper()
	settings := readClaudeSettings(t, path)
	env, ok := settings["env"].(map[string]any)
	if !ok {
		return "", false
	}
	val, ok := env[claudeEnvKey].(string)
	return val, ok
}

func TestParseTargets(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{name: "both", input: "claude,codex", want: []string{"claude", "codex"}},
		{name: "single", input: "claude", want: []string{"claude"}},
		{name: "spaces and case", input: " Claude , CODEX ", want: []string{"claude", "codex"}},
		{name: "dedup", input: "claude,claude", want: []string{"claude"}},
		{name: "unknown", input: "claude,cursor", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTargets(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestManager_AttachDetach_RoundTrip(t *testing.T) {
	m := testManager(t, TargetClaude, TargetCodex)

	originalSettings := `{
  "model": "opus",
  "permissions": {
    "allow": ["WebFetch"]
  }
}
`
	originalCodex := "model = \"gpt-5.5\"\nnotify = [\"/bin/true\"]\n\n[tui]\nstatus_line = [\"model\"]\n"
	require.NoError(t, os.WriteFile(m.claudeSettingsPath, []byte(originalSettings), 0o644))
	require.NoError(t, os.WriteFile(m.codexConfigPath, []byte(originalCodex), 0o600))

	require.NoError(t, m.Attach())

	// Claude: env override merged, existing keys preserved.
	settings := readClaudeSettings(t, m.claudeSettingsPath)
	assert.Equal(t, "opus", settings["model"])
	assert.Contains(t, settings, "permissions")
	url, ok := claudeBaseURL(t, m.claudeSettingsPath)
	require.True(t, ok)
	assert.Equal(t, testProxyURL, url)

	// Codex: block prepended, original content intact, /v1 suffix applied.
	codexData, err := os.ReadFile(m.codexConfigPath)
	require.NoError(t, err)
	assert.True(t, len(codexData) > len(originalCodex))
	assert.Contains(t, string(codexData), `openai_base_url = "http://localhost:4567/v1"`)
	assert.Contains(t, string(codexData), originalCodex)

	// Codex file keeps its restrictive permissions.
	info, err := os.Stat(m.codexConfigPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	// State file written.
	_, err = os.Stat(m.statePath)
	require.NoError(t, err)

	require.NoError(t, m.Detach())

	// Claude: env key gone, everything else preserved.
	settings = readClaudeSettings(t, m.claudeSettingsPath)
	assert.Equal(t, "opus", settings["model"])
	assert.NotContains(t, settings, "env")

	// Codex: byte-identical restore.
	codexData, err = os.ReadFile(m.codexConfigPath)
	require.NoError(t, err)
	assert.Equal(t, originalCodex, string(codexData))

	// State file removed.
	_, err = os.Stat(m.statePath)
	assert.True(t, os.IsNotExist(err))
}

func TestManager_Attach_CreatesMissingFiles(t *testing.T) {
	m := testManager(t, TargetClaude, TargetCodex)

	require.NoError(t, m.Attach())

	url, ok := claudeBaseURL(t, m.claudeSettingsPath)
	require.True(t, ok)
	assert.Equal(t, testProxyURL, url)
	_, err := os.Stat(m.codexConfigPath)
	require.NoError(t, err)

	require.NoError(t, m.Detach())

	// Files mirra created are removed once they hold nothing else.
	_, err = os.Stat(m.claudeSettingsPath)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(m.codexConfigPath)
	assert.True(t, os.IsNotExist(err))
}

func TestManager_Attach_SkipsMissingTools(t *testing.T) {
	m := testManager(t, TargetClaude, TargetCodex)
	require.NoError(t, os.RemoveAll(filepath.Dir(m.codexConfigPath)))

	require.NoError(t, m.Attach())

	st, err := m.loadState()
	require.NoError(t, err)
	require.NotNil(t, st)
	assert.Contains(t, st.Tools, TargetClaude)
	assert.NotContains(t, st.Tools, TargetCodex)

	require.NoError(t, m.Detach())
}

func TestManager_Attach_FailsWhenNothingAttaches(t *testing.T) {
	m := testManager(t, TargetClaude)
	require.NoError(t, os.RemoveAll(filepath.Dir(m.claudeSettingsPath)))

	assert.Error(t, m.Attach())
}

func TestManager_Attach_RepairsStaleState(t *testing.T) {
	// First run attaches and then "crashes" (no Detach).
	m1 := testManager(t, TargetClaude, TargetCodex)
	original := "model = \"gpt-5.5\"\n"
	require.NoError(t, os.WriteFile(m1.codexConfigPath, []byte(original), 0o600))
	require.NoError(t, m1.Attach())

	// Second run on another port reuses the same paths and state file.
	m2 := &Manager{
		proxyURL:           "http://localhost:9999",
		targets:            []string{TargetClaude, TargetCodex},
		log:                slog.Default(),
		claudeSettingsPath: m1.claudeSettingsPath,
		codexConfigPath:    m1.codexConfigPath,
		statePath:          m1.statePath,
	}
	require.NoError(t, m2.Attach())

	// Overrides point at the new proxy and are not stacked.
	url, ok := claudeBaseURL(t, m2.claudeSettingsPath)
	require.True(t, ok)
	assert.Equal(t, "http://localhost:9999", url)
	codexData, err := os.ReadFile(m2.codexConfigPath)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(codexData), codexBaseURLKey))

	// The first (dead) instance's Detach must not undo the new attachment.
	require.NoError(t, m1.Detach())
	url, ok = claudeBaseURL(t, m2.claudeSettingsPath)
	require.True(t, ok)
	assert.Equal(t, "http://localhost:9999", url)

	require.NoError(t, m2.Detach())
	codexData, err = os.ReadFile(m2.codexConfigPath)
	require.NoError(t, err)
	assert.Equal(t, original, string(codexData))
}

func TestManager_Detach_NoopWithoutAttach(t *testing.T) {
	m := testManager(t, TargetClaude)
	require.NoError(t, m.Detach())
}

func TestRestoreViaState(t *testing.T) {
	// Simulates `mirra detach` after a crash: a fresh manager restores purely
	// from the on-disk state file.
	m := testManager(t, TargetClaude, TargetCodex)
	require.NoError(t, m.Attach())

	m2 := &Manager{
		proxyURL:  "",
		log:       slog.Default(),
		statePath: m.statePath,
	}
	st, err := m2.loadState()
	require.NoError(t, err)
	require.NotNil(t, st)
	m2.restore(st)
	require.NoError(t, os.Remove(m2.statePath))

	_, err = os.Stat(m.claudeSettingsPath)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(m.codexConfigPath)
	assert.True(t, os.IsNotExist(err))
}
