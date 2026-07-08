package attach

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Codex reads openai_base_url from config.toml as a base URL override for its
// built-in openai provider, in both API-key and ChatGPT-subscription auth
// modes. The override lives in a marker-delimited block so detach can remove
// exactly what attach added.
const (
	codexBaseURLKey       = "openai_base_url"
	codexMarkerBegin      = "# >>> mirra attach >>> managed by `mirra start --attach`; do not edit"
	codexMarkerEnd        = "# <<< mirra attach <<<"
	codexMarkerBeginMatch = "# >>> mirra attach >>>"
	codexMarkerEndMatch   = "# <<< mirra attach <<<"
)

// attachCodex prepends the override block to Codex's config.toml. Top-level
// TOML keys must appear before any [table] header, so the block can only go
// at the top of the file.
func attachCodex(configPath, baseURL string) (*toolState, error) {
	if _, err := os.Stat(filepath.Dir(configPath)); err != nil {
		return nil, ErrNotInstalled
	}

	var content string
	created := false
	data, err := os.ReadFile(configPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		created = true
	case err != nil:
		return nil, err
	default:
		content = string(data)
	}

	// A leftover block from a run that lost its state file must not stack.
	content, _ = stripCodexBlock(content)

	if line, found := findTopLevelKey(content, codexBaseURLKey); found {
		return nil, fmt.Errorf("%s already sets %s (line %d); remove it to let mirra manage codex", configPath, codexBaseURLKey, line)
	}

	block := fmt.Sprintf("%s\n%s = %q\n%s\n\n", codexMarkerBegin, codexBaseURLKey, baseURL, codexMarkerEnd)
	ts := &toolState{ConfigPath: configPath, CreatedFile: created}
	return ts, writeFileAtomic(configPath, []byte(block+content), fileModeOr(configPath, 0o600))
}

// detachCodex removes the override block, restoring the file's prior content.
func detachCodex(ts *toolState) error {
	data, err := os.ReadFile(ts.ConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	remaining, found := stripCodexBlock(string(data))
	if !found {
		return nil
	}
	if ts.CreatedFile && strings.TrimSpace(remaining) == "" {
		return os.Remove(ts.ConfigPath)
	}
	return writeFileAtomic(ts.ConfigPath, []byte(remaining), fileModeOr(ts.ConfigPath, 0o600))
}

// stripCodexBlock removes the mirra marker block plus the blank line attach
// inserted after it. An unterminated block (end marker deleted) is left
// untouched rather than risking removal of user content.
func stripCodexBlock(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	inBlock, found := false, false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case !inBlock && strings.HasPrefix(trimmed, codexMarkerBeginMatch):
			inBlock, found = true, true
		case inBlock && strings.HasPrefix(trimmed, codexMarkerEndMatch):
			inBlock = false
		case !inBlock:
			out = append(out, line)
		}
	}
	if inBlock {
		return content, false
	}
	result := strings.Join(out, "\n")
	result = strings.TrimPrefix(result, "\n")
	return result, found
}

// findTopLevelKey reports whether key is assigned in the top-level section of
// a TOML document, i.e. before the first [table] header. Assignments inside
// tables do not conflict with a top-level insertion.
func findTopLevelKey(content, key string) (int, bool) {
	for i, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			break
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if rest, ok := strings.CutPrefix(trimmed, key); ok {
			if rest = strings.TrimSpace(rest); strings.HasPrefix(rest, "=") {
				return i + 1, true
			}
		}
	}
	return 0, false
}
