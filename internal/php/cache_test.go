package php

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

var sampleBranches = []Branch{
	{Name: "8.4", Latest: "8.4.20", Status: StatusSupported},
	{Name: "7.4", Latest: "7.4.33", Status: StatusEOL},
}

func TestCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()

	if _, ok := readCache(dir); ok {
		t.Fatal("readCache on empty dir should miss")
	}

	if err := writeCache(dir, sampleBranches); err != nil {
		t.Fatal(err)
	}

	got, ok := readCache(dir)
	if !ok {
		t.Fatal("readCache should hit right after writeCache")
	}
	if !reflect.DeepEqual(got, sampleBranches) {
		t.Errorf("readCache = %+v, want %+v", got, sampleBranches)
	}
}

func TestCacheExpired(t *testing.T) {
	dir := t.TempDir()
	writeRawCache(t, dir, branchCache{
		CachedAt: time.Now().Add(-cacheTTL - time.Minute),
		Branches: sampleBranches,
	})

	if _, ok := readCache(dir); ok {
		t.Error("readCache should miss when cache is older than TTL")
	}
}

func TestCacheCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := availableCacheFile(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, ok := readCache(dir); ok {
		t.Error("readCache should miss on corrupt JSON")
	}
}

func TestFetchAllBranchesCachedUsesFreshCache(t *testing.T) {
	dir := t.TempDir()
	writeRawCache(t, dir, branchCache{CachedAt: time.Now(), Branches: sampleBranches})

	// A cancelled context guarantees any network call would fail,
	// so success proves the result came from the cache.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := FetchAllBranchesCached(ctx, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, sampleBranches) {
		t.Errorf("FetchAllBranchesCached = %+v, want %+v", got, sampleBranches)
	}

	if _, err := FetchAllBranchesCached(ctx, dir, true); err == nil {
		t.Error("forceRefresh should bypass the cache and hit the network")
	}
}

func writeRawCache(t *testing.T, dir string, c branchCache) {
	t.Helper()
	path := availableCacheFile(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
