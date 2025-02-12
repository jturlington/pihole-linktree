package health

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"pihole-linktree/pkg/cache"
	"pihole-linktree/pkg/config"
)

type Status struct {
	Status      string `json:"status"`
	PiholeHost  string `json:"pihole_host"`
	CacheStatus string `json:"cache_status"`
}

func Handler(cache *cache.Cache, cfg *config.Config) http.HandlerFunc {
	status := Status{
		Status:      "healthy",
		PiholeHost:  "unreachable",
		CacheStatus: "empty",
	}

	// Check if cache has data
	if info := cache.Get(); info != nil {
		status.CacheStatus = fmt.Sprintf("loaded with %d records", len(info.Subdomains))
	}

	// Try to resolve Pi-hole host
	if ips, err := net.LookupHost(cfg.PiholeHost); err == nil && len(ips) > 0 {
		status.PiholeHost = "reachable"
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(status)
	}
}
