package util

import "os"

func GetEnvDefault(key, defaultValue string) string {
	env := os.Getenv(key)
	if env == "" {
		return defaultValue
	}

	return env
}
