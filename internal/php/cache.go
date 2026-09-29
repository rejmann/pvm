package php

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const cacheTTL = 24 * time.Hour

type branchCache struct {
	CachedAt time.Time `json:"cached_at"`
	Branches []Branch  `json:"branches"`
}

func readCache(path string) ([]Branch, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var c branchCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, false
	}
	if time.Since(c.CachedAt) > cacheTTL {
		return nil, false
	}
	// Caches written before branches were sorted keep a random order.
	sortBranches(c.Branches)
	return c.Branches, true
}

func writeCache(path string, branches []Branch) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(branchCache{
		CachedAt: time.Now(),
		Branches: branches,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// FetchAllBranchesCached is FetchAllBranches, cached for a day in cacheFile.
func FetchAllBranchesCached(ctx context.Context, cacheFile string, forceRefresh bool) ([]Branch, error) {
	if !forceRefresh {
		if branches, ok := readCache(cacheFile); ok {
			return branches, nil
		}
	}
	branches, err := FetchAllBranches(ctx)
	if err != nil {
		return nil, err
	}
	_ = writeCache(cacheFile, branches)
	return branches, nil
}
