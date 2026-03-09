package config

// Config holds application configuration.
type Config struct {
	AppDataDir  string
	Port        int
	MaxUsers    int
	StorageRoot string
}

// Load applies env/config overrides to cfg. For MVP we do not read a config file.
func Load(cfg *Config) (*Config, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	return cfg, nil
}
