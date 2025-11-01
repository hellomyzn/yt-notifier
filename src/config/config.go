package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	DefaultOutput    string
	CategoryToOutput map[string]string
	CategoryToEnv    map[string]string
	RateLimit        struct {
		FetchSleepMS int
		PostSleepMS  int
	}
	Filters struct {
		IncludePremieres bool
		IncludeLive      bool
		IncludeShorts    bool
	}
	Timezone string
}

func Load(path string) (*AppConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &AppConfig{
		CategoryToOutput: map[string]string{},
		CategoryToEnv:    map[string]string{},
	}

	scanner := bufio.NewScanner(f)
	section := ""
	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if !strings.HasPrefix(raw, " ") && !strings.HasPrefix(raw, "\t") {
			key, value, hasValue := splitKeyValue(trimmed)
			if !hasValue {
				section = key
				continue
			}
			section = ""
			if err := applyTopLevel(cfg, key, value); err != nil {
				return nil, err
			}
			continue
		}

		if section == "" {
			continue
		}
		key, value, hasValue := splitKeyValue(trimmed)
		if !hasValue {
			continue
		}
		if err := applySection(cfg, section, key, value); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func splitKeyValue(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return strings.TrimSpace(line), "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if c := strings.Index(value, "#"); c != -1 {
		value = strings.TrimSpace(value[:c])
	}
	return key, trimQuotes(value), true
}

func applyTopLevel(cfg *AppConfig, key, value string) error {
	switch key {
	case "default_output":
		cfg.DefaultOutput = value
	case "timezone":
		cfg.Timezone = value
	default:
		return nil
	}
	return nil
}

func applySection(cfg *AppConfig, section, key, value string) error {
	switch section {
	case "category_to_output":
		cfg.CategoryToOutput[key] = value
	case "category_to_env":
		cfg.CategoryToEnv[key] = value
	case "rate_limit":
		iv, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid int for %s: %w", key, err)
		}
		switch key {
		case "fetch_sleep_ms":
			cfg.RateLimit.FetchSleepMS = iv
		case "post_sleep_ms":
			cfg.RateLimit.PostSleepMS = iv
		}
	case "filters":
		bv, err := strconv.ParseBool(strings.ToLower(value))
		if err != nil {
			return fmt.Errorf("invalid bool for %s: %w", key, err)
		}
		switch key {
		case "include_premieres":
			cfg.Filters.IncludePremieres = bv
		case "include_live":
			cfg.Filters.IncludeLive = bv
		case "include_shorts":
			cfg.Filters.IncludeShorts = bv
		}
	}
	return nil
}

func trimQuotes(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) || (strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'")) {
			return v[1 : len(v)-1]
		}
	}
	return v
}
