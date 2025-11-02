package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadWebhookFile loads webhook secrets from a simple KEY=VALUE formatted file.
func LoadWebhookFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	secrets := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := parseSecretLine(line)
		if !ok {
			return nil, fmt.Errorf("invalid webhook secret line: %s", line)
		}
		secrets[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return secrets, nil
}

func parseSecretLine(line string) (string, string, bool) {
	idx := strings.Index(line, "=")
	if idx <= 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	if c := strings.Index(value, "#"); c != -1 {
		value = strings.TrimSpace(value[:c])
	}
	if value == "" {
		return "", "", false
	}
	return key, trimQuotes(value), true
}
