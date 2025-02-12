package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

type DomainInfo struct {
	Domain     string      `json:"domain"`
	Subdomains []SubRecord `json:"subdomains"`
}

type SubRecord struct {
	Name        string   `json:"name"`
	Title       string   `json:"title,omitempty"`
	ARecords    []string `json:"a_records,omitempty"`
	AAAARecords []string `json:"aaaa_records,omitempty"`
}

type PiholeResponse struct {
	Data [][]string `json:"data"`
}

type Config struct {
	PiholeHost string
	AuthToken  string
	BaseDomain string
	cache      atomic.Pointer[DomainInfo] // Thread-safe cache
}

type HealthStatus struct {
	Status      string `json:"status"`
	PiholeHost  string `json:"pihole_host"`
	CacheStatus string `json:"cache_status"`
}

func loadConfig() (*Config, error) {
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

	return &Config{
		PiholeHost: host,
		AuthToken:  token,
		BaseDomain: domain,
	}, nil
}

func fetchPiholeRecords(cfg *Config) (*DomainInfo, error) {
	// Create a single reusable HTTP client for title fetching
	titleClient := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: true,
		},
	}

	// Create custom transport with longer timeouts
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	url := fmt.Sprintf("https://%s/admin/api.php?customdns&action=get&auth=%s",
		cfg.PiholeHost, cfg.AuthToken)

	log.Printf("Making HTTP request to Pi-hole API")
	start := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Pi-hole: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Received response from Pi-hole in %v", time.Since(start))

	// Move response body reading to a separate function to ensure proper cleanup
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close() // Explicitly close the body
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var piholeResp PiholeResponse
	if err := json.Unmarshal(body, &piholeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	info := &DomainInfo{
		Domain: cfg.BaseDomain,
	}

	// Convert Pi-hole records to our format
	recordCount := 0
	for _, record := range piholeResp.Data {
		if len(record) != 2 {
			log.Printf("Skipping invalid record format: %v", record)
			continue
		}

		title := record[0]
		// Only fetch title if the domain is not an IP address
		if !isIPAddress(record[0]) {
			title = fetchTitle(titleClient, record[0])
		}

		subRecord := SubRecord{
			Name:     record[0],
			Title:    title,
			ARecords: []string{record[1]},
		}
		info.Subdomains = append(info.Subdomains, subRecord)
		recordCount++
	}

	log.Printf("Processed %d DNS records from Pi-hole", recordCount)
	return info, nil
}

// Add these new helper functions
func isIPAddress(s string) bool {
	return net.ParseIP(s) != nil
}

func fetchTitle(client *http.Client, domain string) string {
	url := fmt.Sprintf("https://%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return domain
	}

	resp, err := client.Do(req)
	if err != nil {
		return domain
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain
	}

	// Limit the amount of data read for title extraction
	limitReader := io.LimitReader(resp.Body, 32*1024) // Only read up to 32KB
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return domain
	}

	// Simple title extraction using string search
	bodyStr := string(body)
	titleStart := strings.Index(bodyStr, "<title>")
	titleEnd := strings.Index(bodyStr, "</title>")
	if titleStart >= 0 && titleEnd > titleStart {
		title := bodyStr[titleStart+7 : titleEnd]
		return strings.TrimSpace(title)
	}

	return domain
}

// Update the cache update function to properly handle old cache entries
func (cfg *Config) updateCache() error {
	info, err := fetchPiholeRecords(cfg)
	if err != nil {
		return fmt.Errorf("cache update failed: %w", err)
	}

	// Get the old cache before storing the new one
	oldCache := cfg.cache.Swap(info)

	// Clear any references in the old cache (optional, but can help GC)
	if oldCache != nil {
		oldCache.Subdomains = nil
	}

	log.Printf("Cache updated successfully with %d subdomains", len(info.Subdomains))
	return nil
}

func (cfg *Config) startCacheUpdater(ctx context.Context) {
	log.Printf("Starting cache updater with 15 second refresh interval")
	ticker := time.NewTicker(15 * time.Second)

	go func() {
		// Initial cache load
		log.Printf("Performing initial cache load...")
		if err := cfg.updateCache(); err != nil {
			log.Printf("Initial cache update failed: %v", err)
		}

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				log.Printf("Cache updater stopped")
				return
			case t := <-ticker.C:
				log.Printf("Running scheduled cache update at %v", t.Format("15:04:05"))
				if err := cfg.updateCache(); err != nil {
					log.Printf("Scheduled cache update failed: %v", err)
				}
			}
		}
	}()
}

func handleHomePage(cfg *Config) http.HandlerFunc {
	// Parse the template at startup
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handling request from %s - %s %s", r.RemoteAddr, r.Method, r.URL.Path)

		info := cfg.cache.Load()
		if info == nil {
			log.Printf("Cache miss - no data available yet")
			http.Error(w, "Data not yet available", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.Execute(w, info); err != nil {
			log.Printf("Error rendering template: %v", err)
			return
		}
		log.Printf("Successfully served response with %d subdomains", len(info.Subdomains))
	}
}

func healthHandler(cfg *Config) http.HandlerFunc {
	status := HealthStatus{
		Status:      "healthy",
		PiholeHost:  "unreachable",
		CacheStatus: "empty",
	}

	// Check if cache has data
	if info := cfg.cache.Load(); info != nil {
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

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("Starting DNS record viewer service")

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Configuration loaded successfully for domain %s", cfg.BaseDomain)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg.startCacheUpdater(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleHomePage(cfg))
	mux.HandleFunc("GET /health", healthHandler(cfg))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		sig := <-sigChan

		log.Printf("Received signal %v, initiating shutdown...", sig)
		cancel() // Stop cache updater

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server listening on :8080")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	log.Printf("Server shutdown complete")
}
