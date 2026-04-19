package config

// Config holds the application configuration
type Config struct {
	LLM      LLMConfig
	Cache    CacheConfig
	Analysis AnalysisConfig
	Output   OutputConfig
}

type LLMConfig struct {
	Provider  string
	Model     string
	MaxTokens int
	Temp      float64
}

type CacheConfig struct {
	Enabled bool
	Dir     string
	TTL     int
}

type AnalysisConfig struct {
	LocalChecks bool
	AIEnabled   bool
	Threshold   float64
}

type OutputConfig struct {
	Format string
}

// LoadConfig loads configuration from files and env vars
func LoadConfig() *Config {
	// TODO: Implement config loading
	return &Config{}
}
