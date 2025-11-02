package config

// LoadWebhookFile loads webhook secrets from a simple KEY=VALUE formatted file.
func LoadWebhookFile(path string) (map[string]string, error) {
	return LoadEnvFile(path)
}
