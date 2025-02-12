package cache

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"pihole-linktree/pkg/pihole"
)

type Cache struct {
	data            atomic.Pointer[pihole.DomainInfo]
	client          *pihole.Client
	refreshInterval time.Duration
	baseDomain      string
}

func New(client *pihole.Client, refreshInterval int, baseDomain string) *Cache {
	return &Cache{
		client:          client,
		refreshInterval: time.Duration(refreshInterval) * time.Second,
		baseDomain:      baseDomain,
	}
}

func (c *Cache) Start(ctx context.Context) {
	log.Printf("Starting cache updater with %v refresh interval", c.refreshInterval)
	ticker := time.NewTicker(c.refreshInterval)

	go func() {
		if err := c.update(); err != nil {
			log.Printf("Initial cache update failed: %v", err)
		}

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				log.Printf("Cache updater stopped")
				return
			case <-ticker.C:
				if err := c.update(); err != nil {
					log.Printf("Cache update failed: %v", err)
				}
			}
		}
	}()
}

func (c *Cache) update() error {
	info, err := c.client.FetchRecords(c.baseDomain)
	if err != nil {
		return err
	}

	oldCache := c.data.Swap(info)
	if oldCache != nil {
		oldCache.Subdomains = nil
	}

	log.Printf("Cache updated successfully with %d subdomains", len(info.Subdomains))
	return nil
}

func (c *Cache) Get() *pihole.DomainInfo {
	return c.data.Load()
}
