package config

import (
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v2"
)

// Config holds the application configuration

type Config struct {
	LLM      LLMConfig      `yaml:"llm"`
	Cache    CacheConfig    `yaml:"cache"`
	Analysis AnalysisConfig `yaml:"analysis"`
	Output   OutputConfig   `yaml:"output"`
}

type LLMConfig struct {
	Provider  string  `yaml:"provider"`
	Model     string  `yaml:"model"`
	MaxTokens int     `yaml:"max_tokens"`
	Temp      float64 `yaml:"temperature"`
}

type CacheConfig struct {
	Enabled bool   `yaml:"enabled"`
	Dir     string `yaml:"dir"`
	TTL     int    `yaml:"ttl"`
}

type AnalysisConfig struct {
	LocalChecks bool    `yaml:"local_checks"`
	AIEnabled   bool    `yaml:"ai_enabled"`
	Threshold   float64 `yaml:"threshold"`
}

type OutputConfig struct {
	Format string `yaml:"format"`
}

// LoadConfig loads configuration from files and env vars (with precedence)
func LoadConfig() *Config {
	cfg := &Config{
		LLM: LLMConfig{
			Provider:  "mistral",
			Model:     "mistral-tiny",
			MaxTokens: 1024,
			Temp:      0.7,
		},
		Cache: CacheConfig{
			Enabled: true,
			Dir:     defaultCacheDir(),
			TTL:     604800, // 7 days
		},
		Analysis: AnalysisConfig{
			LocalChecks: true,
			AIEnabled:   true,
			Threshold:   0.5,
		},
		Output: OutputConfig{
			Format: "json",
		},
	}

	// 1. Load from YAML file (if exists)
	yamlPaths := []string{
		".code-review-agent.yml",
		filepath.Join(homeDir(), ".code-review-agent.yml"),
	}
	for _, path := range yamlPaths {
		if data, err := os.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(data, cfg); err == nil {
				break
			}
		}
	}

	// 2. Override with environment variables
	overlayEnv(cfg)

	return cfg
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return ""
}

func defaultCacheDir() string {
	h := homeDir()
	if h != "" {
		return filepath.Join(h, ".cache", "code-review-agent")
	}
	return filepath.Join(os.TempDir(), "code-review-agent-cache")
}

func overlayEnv(cfg *Config) {
	if v := os.Getenv("REVIEW_LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("REVIEW_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("REVIEW_LLM_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LLM.MaxTokens = n
		}
	}
	if v := os.Getenv("REVIEW_LLM_TEMP"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.LLM.Temp = f
		}
	}
	if v := os.Getenv("REVIEW_CACHE_ENABLED"); v != "" {
		cfg.Cache.Enabled = v == "true"
	}
	if v := os.Getenv("REVIEW_CACHE_DIR"); v != "" {
		cfg.Cache.Dir = v
	}
	if v := os.Getenv("REVIEW_CACHE_TTL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Cache.TTL = n
		}
	}
	if v := os.Getenv("REVIEW_ANALYSIS_LOCAL_CHECKS"); v != "" {
		cfg.Analysis.LocalChecks = v == "true"
	}
	if v := os.Getenv("REVIEW_ANALYSIS_AI_ENABLED"); v != "" {
		cfg.Analysis.AIEnabled = v == "true"
	}
	if v := os.Getenv("REVIEW_ANALYSIS_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Analysis.Threshold = f
		}
	}
	if v := os.Getenv("REVIEW_OUTPUT_FORMAT"); v != "" {
		cfg.Output.Format = v
	}
}
