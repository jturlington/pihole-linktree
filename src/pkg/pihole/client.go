package pihole

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	host      string
	authToken string
	client    *http.Client
}

type PiholeResponse struct {
	Data [][]string `json:"data"`
}

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

func NewClient(host, authToken string) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Client{
		host:      host,
		authToken: authToken,
		client: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
	}
}

func (c *Client) FetchRecords(baseDomain string) (*DomainInfo, error) {
	url := fmt.Sprintf("https://%s/admin/api.php?customdns&action=get&auth=%s",
		c.host, c.authToken)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Pi-hole: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var piholeResp PiholeResponse
	if err := json.Unmarshal(body, &piholeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	info := &DomainInfo{
		Domain: baseDomain,
	}

	titleClient := createTitleClient()

	for _, record := range piholeResp.Data {
		if len(record) != 2 {
			log.Printf("Skipping invalid record format: %v", record)
			continue
		}

		title := record[0]
		if !isIPAddress(record[0]) {
			title = fetchTitle(titleClient, record[0])
		}

		subRecord := SubRecord{
			Name:     record[0],
			Title:    title,
			ARecords: []string{record[1]},
		}
		info.Subdomains = append(info.Subdomains, subRecord)
	}

	return info, nil
}

// Helper functions moved from main.go
func createTitleClient() *http.Client {
	return &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: true,
		},
	}
}

func isIPAddress(s string) bool {
	return net.ParseIP(s) != nil
}

func fetchTitle(client *http.Client, domain string) string {
	// Try common paths in order
	paths := []string{
		"",           // Root path
		"/home",      // Common home path
		"/admin",     // Admin path
		"/dashboard", // Dashboard path
		"/index",     // Index path
	}

	for _, path := range paths {
		url := fmt.Sprintf("https://%s%s", domain, path)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		// Limit the amount of data read for title extraction
		limitReader := io.LimitReader(resp.Body, 32*1024) // Only read up to 32KB
		body, err := io.ReadAll(limitReader)
		if err != nil {
			continue
		}

		// Simple title extraction using string search
		bodyStr := string(body)
		titleStart := strings.Index(bodyStr, "<title>")
		titleEnd := strings.Index(bodyStr, "</title>")
		if titleStart >= 0 && titleEnd > titleStart {
			title := bodyStr[titleStart+7 : titleEnd]
			return strings.TrimSpace(title)
		}
	}

	return domain
}
