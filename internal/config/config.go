package config

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config holds the application configuration
type Config struct {
	LLM      LLMConfig     `yaml:"llm"`
	Cache    CacheConfig   `yaml:"cache"`
	Analysis AnalysisConfig `yaml:"analysis"`
	Output   OutputConfig  `yaml:"output"`
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

func overlayEnv(cfg *Config) {
	// LLM config
	if val := os.Getenv("CODE_REVIEW_LLM_PROVIDER"); val != "" {
		cfg.LLM.Provider = val
	}
	if val := os.Getenv("CODE_REVIEW_LLM_MODEL"); val != "" {
		cfg.LLM.Model = val
	}
	if val := os.Getenv("CODE_REVIEW_LLM_MAX_TOKENS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			cfg.LLM.MaxTokens = n
		}
	}
	if val := os.Getenv("CODE_REVIEW_LLM_TEMP"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.LLM.Temp = f
		}
	}

	// Cache config
	if val := os.Getenv("CODE_REVIEW_CACHE_ENABLED"); val != "" {
		cfg.Cache.Enabled = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("CODE_REVIEW_CACHE_DIR"); val != "" {
		cfg.Cache.Dir = val
	}
	if val := os.Getenv("CODE_REVIEW_CACHE_TTL"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			cfg.Cache.TTL = n
		}
	}

	// Analysis config
	if val := os.Getenv("CODE_REVIEW_ANALYSIS_LOCAL"); val != "" {
		cfg.Analysis.LocalChecks = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("CODE_REVIEW_ANALYSIS_AI"); val != "" {
		cfg.Analysis.AIEnabled = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("CODE_REVIEW_ANALYSIS_THRESHOLD"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.Analysis.Threshold = f
		}
	}

	// Output config
	if val := os.Getenv("CODE_REVIEW_OUTPUT_FORMAT"); val != "" {
		cfg.Output.Format = val
	}
}

func defaultCacheDir() string {
	if h := homeDir(); h != "" {
		return filepath.Join(h, ".cache", "code-review-agent")
	}
	return "/tmp/code-review-agent"
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}
	return ""
}
