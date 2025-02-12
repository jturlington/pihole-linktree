package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	PiholeHost           string
	AuthToken            string
	BaseDomain           string
	CacheRefreshInterval int
}

func Load() (*Config, error) {
	host := os.Getenv("PIHOLE_HOST")
	if host == "" {
		return nil, fmt.Errorf("PIHOLE_HOST environment variable is required")
	}

	token := os.Getenv("PIHOLE_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("PIHOLE_TOKEN environment variable is required")
	}

	domain := os.Getenv("BASE_DOMAIN")
	if domain == "" {
		return nil, fmt.Errorf("BASE_DOMAIN environment variable is required")
	}

	refreshInterval, err := strconv.Atoi(os.Getenv("CACHE_REFRESH_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("CACHE_REFRESH_INTERVAL environment variable must be an integer")
	}

	return &Config{
		PiholeHost:           host,
		AuthToken:            token,
		BaseDomain:           domain,
		CacheRefreshInterval: refreshInterval,
	}, nil
}
