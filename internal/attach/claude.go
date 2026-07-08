package attach

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const claudeEnvKey = "ANTHROPIC_BASE_URL"

// attachClaude merges env.ANTHROPIC_BASE_URL into Claude Code's
// settings.json. Claude Code applies the env block to every new session, so
// sessions started while attached route their API traffic through the proxy.
func attachClaude(settingsPath, proxyURL string) (*toolState, error) {
	if _, err := os.Stat(filepath.Dir(settingsPath)); err != nil {
		return nil, ErrNotInstalled
	}

	settings := map[string]any{}
	created := false
	data, err := os.ReadFile(settingsPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		created = true
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(data, &settings); err != nil {
			return nil, fmt.Errorf("parse %s: %w", settingsPath, err)
		}
	}

	ts := &toolState{ConfigPath: settingsPath, CreatedFile: created}

	env, ok := settings["env"].(map[string]any)
	if !ok {
		if _, exists := settings["env"]; exists {
			return nil, fmt.Errorf(`%s: "env" is not an object`, settingsPath)
		}
		env = map[string]any{}
		settings["env"] = env
		ts.CreatedEnv = true
	}

	if prev, exists := env[claudeEnvKey]; exists {
		prevStr, ok := prev.(string)
		if !ok {
			return nil, fmt.Errorf("%s: env.%s is not a string", settingsPath, claudeEnvKey)
		}
		// A value equal to proxyURL is a leftover from an unclean run of this
		// same proxy, not a user setting worth restoring.
		if prevStr != proxyURL {
			ts.HadPrevValue = true
			ts.PrevValue = prevStr
		}
	}
	env[claudeEnvKey] = proxyURL

	return ts, writeJSONFile(settingsPath, settings)
}

// detachClaude removes (or restores the prior value of) the env override. A
// value that no longer matches what mirra wrote means the user changed it
// while attached, and it is left alone.
func detachClaude(ts *toolState, proxyURL string) error {
	data, err := os.ReadFile(ts.ConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	settings := map[string]any{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parse %s: %w", ts.ConfigPath, err)
	}

	env, ok := settings["env"].(map[string]any)
	if !ok {
		return nil
	}
	current, exists := env[claudeEnvKey]
	if !exists {
		return nil
	}
	if cur, ok := current.(string); !ok || cur != proxyURL {
		slog.Warn("env."+claudeEnvKey+" was changed while attached; leaving it in place", "path", ts.ConfigPath)
		return nil
	}

	if ts.HadPrevValue {
		env[claudeEnvKey] = ts.PrevValue
	} else {
		delete(env, claudeEnvKey)
		if ts.CreatedEnv && len(env) == 0 {
			delete(settings, "env")
		}
	}

	if ts.CreatedFile && len(settings) == 0 {
		return os.Remove(ts.ConfigPath)
	}
	return writeJSONFile(ts.ConfigPath, settings)
}

func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, fileModeOr(path, 0o644))
}
