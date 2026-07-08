package attach

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func claudePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "settings.json")
}

func TestAttachClaude(t *testing.T) {
	tests := []struct {
		name     string
		existing string // empty means no file
		wantErr  string
	}{
		{
			name: "no settings file",
		},
		{
			name:     "existing settings without env",
			existing: `{"model": "opus"}`,
		},
		{
			name:     "existing env with other vars",
			existing: `{"env": {"FOO": "bar"}}`,
		},
		{
			name:     "env is not an object",
			existing: `{"env": "nope"}`,
			wantErr:  `"env" is not an object`,
		},
		{
			name:     "existing base url is not a string",
			existing: `{"env": {"ANTHROPIC_BASE_URL": 42}}`,
			wantErr:  "is not a string",
		},
		{
			name:     "invalid json",
			existing: `{`,
			wantErr:  "parse",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := claudePath(t)
			if tt.existing != "" {
				require.NoError(t, os.WriteFile(path, []byte(tt.existing), 0o644))
			}

			ts, err := attachClaude(path, testProxyURL)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, path, ts.ConfigPath)

			url, ok := claudeBaseURL(t, path)
			require.True(t, ok)
			assert.Equal(t, testProxyURL, url)
		})
	}
}

func TestAttachClaude_NotInstalled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "settings.json")
	_, err := attachClaude(path, testProxyURL)
	assert.ErrorIs(t, err, ErrNotInstalled)
}

func TestAttachClaude_RestoresPreviousValue(t *testing.T) {
	path := claudePath(t)
	require.NoError(t, os.WriteFile(path, []byte(`{"env": {"ANTHROPIC_BASE_URL": "http://other-proxy"}}`), 0o644))

	ts, err := attachClaude(path, testProxyURL)
	require.NoError(t, err)
	assert.True(t, ts.HadPrevValue)
	assert.Equal(t, "http://other-proxy", ts.PrevValue)

	require.NoError(t, detachClaude(ts, testProxyURL))
	url, ok := claudeBaseURL(t, path)
	require.True(t, ok)
	assert.Equal(t, "http://other-proxy", url)
}

func TestAttachClaude_LeftoverValueNotTreatedAsPrevious(t *testing.T) {
	// A crash can leave the proxy URL in settings.json with no state file.
	// Re-attaching must not "remember" it as a user value.
	path := claudePath(t)
	require.NoError(t, os.WriteFile(path, []byte(`{"env": {"ANTHROPIC_BASE_URL": "`+testProxyURL+`"}}`), 0o644))

	ts, err := attachClaude(path, testProxyURL)
	require.NoError(t, err)
	assert.False(t, ts.HadPrevValue)

	require.NoError(t, detachClaude(ts, testProxyURL))
	_, ok := claudeBaseURL(t, path)
	assert.False(t, ok)
}

func TestDetachClaude_LeavesUserChangedValue(t *testing.T) {
	path := claudePath(t)
	ts, err := attachClaude(path, testProxyURL)
	require.NoError(t, err)

	// User points the env var somewhere else while attached.
	require.NoError(t, os.WriteFile(path, []byte(`{"env": {"ANTHROPIC_BASE_URL": "http://user-choice"}}`), 0o644))

	require.NoError(t, detachClaude(ts, testProxyURL))
	url, ok := claudeBaseURL(t, path)
	require.True(t, ok)
	assert.Equal(t, "http://user-choice", url)
}

func TestDetachClaude_KeepsEnvWhenNotCreated(t *testing.T) {
	path := claudePath(t)
	require.NoError(t, os.WriteFile(path, []byte(`{"env": {"FOO": "bar"}}`), 0o644))

	ts, err := attachClaude(path, testProxyURL)
	require.NoError(t, err)
	require.NoError(t, detachClaude(ts, testProxyURL))

	settings := readClaudeSettings(t, path)
	env, ok := settings["env"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "bar", env["FOO"])
	assert.NotContains(t, env, claudeEnvKey)
}

func TestDetachClaude_MissingFileIsNoop(t *testing.T) {
	path := claudePath(t)
	ts, err := attachClaude(path, testProxyURL)
	require.NoError(t, err)
	require.NoError(t, os.Remove(path))

	require.NoError(t, detachClaude(ts, testProxyURL))
}
