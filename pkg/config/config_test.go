package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("NAVI_TRANSMISSION_RPC", "http://example.test:9091/transmission/rpc")
	t.Setenv("NAVI_MEDIA_ROOT", "/media/library")
	t.Setenv("NAVI_SMB_PREFIX", "smb://example-nas/media/")
	t.Setenv("NAVI_CATALOG_PATH", "output/catalog.json")
	t.Setenv("NAVI_TMDB_READ_ACCESS_TOKEN", "test-token")
	t.Setenv("NAVI_TMDB_ENRICH_UNMAPPED", "true")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TransmissionRPC != "http://example.test:9091/transmission/rpc" || cfg.CatalogPath != "output/catalog.json" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if cfg.TMDBReadAccessToken != "test-token" || !cfg.TMDBEnrichUnmapped {
		t.Fatalf("TMDB environment values were not applied: %#v", cfg)
	}
}

func TestLoadTMDBTokenFromJSON(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
		"transmission_rpc":"http://example.test/transmission/rpc",
		"tmdb_read_access_token":"file-token",
		"catalog_path":"catalog.json",
		"media_root":"/media/library",
		"smb_prefix":"smb://example-nas/media/"
	}`)
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(filename)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TMDBReadAccessToken != "file-token" || cfg.TMDBLanguage != "en-US" {
		t.Fatalf("unexpected TMDB config: %#v", cfg)
	}
}
