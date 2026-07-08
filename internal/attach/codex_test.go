package attach

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCodexBaseURL = testProxyURL + "/v1"

func codexPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.toml")
}

func TestAttachCodex_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		existing string // empty means no file
	}{
		{
			name: "no config file",
		},
		{
			name:     "top-level keys only",
			existing: "model = \"gpt-5.5\"\nmodel_reasoning_effort = \"high\"\n",
		},
		{
			name:     "top-level keys and tables",
			existing: "model = \"gpt-5.5\"\n\n[projects.\"/tmp/x\"]\ntrust_level = \"trusted\"\n\n[tui]\nstatus_line = [\"model\"]\n",
		},
		{
			name:     "key inside a table does not conflict",
			existing: "[profiles.work]\nopenai_base_url = \"http://elsewhere\"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := codexPath(t)
			if tt.existing != "" {
				require.NoError(t, os.WriteFile(path, []byte(tt.existing), 0o600))
			}

			ts, err := attachCodex(path, testCodexBaseURL)
			require.NoError(t, err)

			data, err := os.ReadFile(path)
			require.NoError(t, err)
			content := string(data)

			// Override block sits at the top, before any table header.
			require.True(t, strings.HasPrefix(content, codexMarkerBegin))
			assert.Contains(t, content, `openai_base_url = "http://localhost:4567/v1"`)
			if tt.existing != "" {
				assert.Contains(t, content, tt.existing)
			}

			require.NoError(t, detachCodex(ts))
			if tt.existing == "" {
				_, err := os.Stat(path)
				assert.True(t, os.IsNotExist(err))
				return
			}
			data, err = os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, tt.existing, string(data))
		})
	}
}

func TestAttachCodex_NotInstalled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "config.toml")
	_, err := attachCodex(path, testCodexBaseURL)
	assert.ErrorIs(t, err, ErrNotInstalled)
}

func TestAttachCodex_RefusesExistingTopLevelOverride(t *testing.T) {
	path := codexPath(t)
	existing := "openai_base_url = \"http://user-proxy\"\nmodel = \"gpt-5.5\"\n"
	require.NoError(t, os.WriteFile(path, []byte(existing), 0o600))

	_, err := attachCodex(path, testCodexBaseURL)
	require.ErrorContains(t, err, "already sets openai_base_url")

	// File untouched on refusal.
	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, existing, string(data))
}

func TestAttachCodex_LeftoverBlockDoesNotStack(t *testing.T) {
	path := codexPath(t)
	original := "model = \"gpt-5.5\"\n"
	require.NoError(t, os.WriteFile(path, []byte(original), 0o600))

	_, err := attachCodex(path, testCodexBaseURL)
	require.NoError(t, err)
	// Second attach (state file lost) replaces the block instead of stacking.
	ts, err := attachCodex(path, "http://localhost:9999/v1")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(data), codexBaseURLKey))
	assert.Contains(t, string(data), "http://localhost:9999/v1")

	require.NoError(t, detachCodex(ts))
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, original, string(data))
}

func TestAttachCodex_UnterminatedBlockIsNotStripped(t *testing.T) {
	// If the user deleted the end marker, stripping would eat their config.
	path := codexPath(t)
	broken := codexMarkerBegin + "\nopenai_base_url = \"http://old\"\nmodel = \"gpt-5.5\"\n"
	require.NoError(t, os.WriteFile(path, []byte(broken), 0o600))

	_, err := attachCodex(path, testCodexBaseURL)
	require.Error(t, err)

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, broken, string(data))
}

func TestDetachCodex_NoBlockIsNoop(t *testing.T) {
	path := codexPath(t)
	original := "model = \"gpt-5.5\"\n"
	require.NoError(t, os.WriteFile(path, []byte(original), 0o600))

	require.NoError(t, detachCodex(&toolState{ConfigPath: path}))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, original, string(data))
}

func TestDetachCodex_MissingFileIsNoop(t *testing.T) {
	require.NoError(t, detachCodex(&toolState{ConfigPath: filepath.Join(t.TempDir(), "config.toml")}))
}

func TestFindTopLevelKey(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{name: "empty", content: "", want: false},
		{name: "present", content: "openai_base_url = \"x\"\n", want: true},
		{name: "indented", content: "  openai_base_url = \"x\"\n", want: true},
		{name: "commented out", content: "# openai_base_url = \"x\"\n", want: false},
		{name: "inside table", content: "[profiles.a]\nopenai_base_url = \"x\"\n", want: false},
		{name: "prefix of longer key", content: "openai_base_url_extra = \"x\"\n", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, found := findTopLevelKey(tt.content, codexBaseURLKey)
			assert.Equal(t, tt.want, found)
		})
	}
}
