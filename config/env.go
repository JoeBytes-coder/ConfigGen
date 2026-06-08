package config

import (
	"os"
	"strings"
)

func LoadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" {
			continue
		}

		if len(val) >= 2 {
			if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
				(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
				val = val[1 : len(val)-1]
			}
		}

		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}

	return nil
}

func GetListenAddr(flagAddr string) string {
	if flagAddr != "" {
		return flagAddr
	}

	if envAddr := os.Getenv("ADDR"); envAddr != "" {
		return envAddr
	}

	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}

	return ":8080"
}
