package config

import (
	"encoding/json"
	"os"
	"strconv"
)

type Config struct {
	Port      int                  `json:"port"`
	Recording RecordingConfig      `json:"recording"`
	Logging   LoggingConfig        `json:"logging"`
	Providers map[string]Provider  `json:"providers"`
}

type RecordingConfig struct {
	Enabled bool   `json:"enabled"`
	Storage string `json:"storage"`
	Path    string `json:"path"`
	Format  string `json:"format"`
}

type LoggingConfig struct {
	Format string `json:"format"` // "pretty", "json", or "plain"
	Level  string `json:"level"`  // "debug", "info", "warn", "error"
}

type Provider struct {
	UpstreamURL string `json:"upstream_url"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Port: 4567,
		Recording: RecordingConfig{
			Enabled: true,
			Storage: "file",
			Path:    "./recordings",
			Format:  "jsonl",
		},
		Logging: LoggingConfig{
			Format: "pretty",
			Level:  "info",
		},
		Providers: map[string]Provider{
			"claude": {UpstreamURL: "https://api.anthropic.com"},
			"openai": {UpstreamURL: "https://api.openai.com"},
		},
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	if port := os.Getenv("TACO_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if enabled := os.Getenv("TACO_RECORDING_ENABLED"); enabled != "" {
		cfg.Recording.Enabled = enabled == "true"
	}

	if recordingPath := os.Getenv("TACO_RECORDING_PATH"); recordingPath != "" {
		cfg.Recording.Path = recordingPath
	}

	if claudeUpstream := os.Getenv("TACO_CLAUDE_UPSTREAM"); claudeUpstream != "" {
		if cfg.Providers == nil {
			cfg.Providers = make(map[string]Provider)
		}
		cfg.Providers["claude"] = Provider{UpstreamURL: claudeUpstream}
	}

	if openaiUpstream := os.Getenv("TACO_OPENAI_UPSTREAM"); openaiUpstream != "" {
		if cfg.Providers == nil {
			cfg.Providers = make(map[string]Provider)
		}
		cfg.Providers["openai"] = Provider{UpstreamURL: openaiUpstream}
	}

	return cfg, nil
}
