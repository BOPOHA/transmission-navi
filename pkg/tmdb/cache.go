package tmdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const cacheVersion = 1
const lookupVersion = 3

type Cache struct {
	Version int                   `json:"version"`
	Entries map[string]CacheEntry `json:"entries"`
}

type CacheEntry struct {
	Found         bool     `json:"found"`
	Metadata      Metadata `json:"metadata,omitempty"`
	UpdatedAt     string   `json:"updated_at"`
	LookupVersion int      `json:"lookup_version,omitempty"`
}

func LoadCache(filename string) (*Cache, error) {
	cache := &Cache{Version: cacheVersion, Entries: make(map[string]CacheEntry)}
	data, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return cache, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read TMDB cache: %w", err)
	}
	if err := json.Unmarshal(data, cache); err != nil {
		return nil, fmt.Errorf("parse TMDB cache: %w", err)
	}
	if cache.Version != cacheVersion {
		return &Cache{Version: cacheVersion, Entries: make(map[string]CacheEntry)}, nil
	}
	if cache.Entries == nil {
		cache.Entries = make(map[string]CacheEntry)
	}
	return cache, nil
}

func (c *Cache) Get(title string, year int, language string, now time.Time) (CacheEntry, bool) {
	entry, ok := c.Entries[cacheKey(title, year, language)]
	if !ok {
		return CacheEntry{}, false
	}
	// Preserve old positive metadata, but retry misses created before the
	// title-variant or resolver fallbacks were introduced.
	if !entry.Found && entry.LookupVersion < lookupVersion {
		return CacheEntry{}, false
	}
	updated, err := time.Parse(time.RFC3339, entry.UpdatedAt)
	if err != nil {
		return CacheEntry{}, false
	}
	ttl := 30 * 24 * time.Hour
	if !entry.Found {
		ttl = 7 * 24 * time.Hour
	}
	if now.Sub(updated) > ttl {
		return CacheEntry{}, false
	}
	return entry, true
}

func (c *Cache) Put(title string, year int, language string, metadata Metadata, found bool, now time.Time) {
	c.Entries[cacheKey(title, year, language)] = CacheEntry{
		Found: found, Metadata: metadata, UpdatedAt: now.UTC().Format(time.RFC3339), LookupVersion: lookupVersion,
	}
}

func (c *Cache) WriteAtomic(filename string) error {
	directory := filepath.Dir(filename)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create TMDB cache directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".tmdb-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary TMDB cache: %w", err)
	}
	name := temporary.Name()
	defer os.Remove(name)
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(c); err != nil {
		temporary.Close()
		return fmt.Errorf("encode TMDB cache: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync TMDB cache: %w", err)
	}
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("set TMDB cache permissions: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close TMDB cache: %w", err)
	}
	if err := os.Rename(name, filename); err != nil {
		return fmt.Errorf("publish TMDB cache: %w", err)
	}
	return nil
}

func cacheKey(title string, year int, language string) string {
	return normalize(title) + "|" + strconv.Itoa(year) + "|" + strings.ToLower(language)
}
