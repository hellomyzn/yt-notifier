package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	DefaultOutput    string            `yaml:"default_output"`
	CategoryToOutput map[string]string `yaml:"category_to_output"`
	CategoryToEnv    map[string]string `yaml:"category_to_env"`
	RateLimit        struct {
		FetchSleepMS int `yaml:"fetch_sleep_ms"`
		PostSleepMS  int `yaml:"post_sleep_ms"`
	} `yaml:"rate_limit"`
	Filters struct {
		IncludePremieres bool `yaml:"include_premieres"`
		IncludeLive      bool `yaml:"include_live"`
		IncludeShorts    bool `yaml:"include_shorts"`
	} `yaml:"filters"`
	Timezone string `yaml:"timezone"`
}

func Load(path string) (*AppConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c AppConfig
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
