package tmdb

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCacheRoundTrip(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	cache, err := LoadCache(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	cache.Put("Movie", 2026, "en-US", Metadata{ID: 42}, true, now)
	filename := filepath.Join(t.TempDir(), "cache", "tmdb.json")
	if err := cache.WriteAtomic(filename); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadCache(filename)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := loaded.Get("Movie", 2026, "en-US", now.Add(time.Hour))
	if !ok || !entry.Found || entry.Metadata.ID != 42 {
		t.Fatalf("unexpected cache entry: %#v, %v", entry, ok)
	}
}

func TestCacheRetriesOldMissAfterLookupUpgrade(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	cache := &Cache{Version: cacheVersion, Entries: map[string]CacheEntry{}}
	key := cacheKey("Nemyslimoe", 2010, "en-US")
	cache.Entries[key] = CacheEntry{Found: false, UpdatedAt: now.Format(time.RFC3339)}
	if _, ok := cache.Get("Nemyslimoe", 2010, "en-US", now); ok {
		t.Fatal("old negative entry should be retried")
	}
	cache.Put("Nemyslimoe", 2010, "en-US", Metadata{}, false, now)
	if _, ok := cache.Get("Nemyslimoe", 2010, "en-US", now); !ok {
		t.Fatal("current negative entry should remain cached")
	}
}
