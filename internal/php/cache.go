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

func availableCacheFile(cacheDir string) string {
	return filepath.Join(cacheDir, "cache", "available.json")
}

func readCache(cacheDir string) ([]Branch, bool) {
	data, err := os.ReadFile(availableCacheFile(cacheDir))
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
	return c.Branches, true
}

func writeCache(cacheDir string, branches []Branch) error {
	path := availableCacheFile(cacheDir)
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

func FetchAllBranchesCached(ctx context.Context, cacheDir string, forceRefresh bool) ([]Branch, error) {
	if !forceRefresh {
		if branches, ok := readCache(cacheDir); ok {
			return branches, nil
		}
	}
	branches, err := FetchAllBranches(ctx)
	if err != nil {
		return nil, err
	}
	_ = writeCache(cacheDir, branches)
	return branches, nil
}
