package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	TransmissionRPC       string `json:"transmission_rpc"`
	TransmissionUsername  string `json:"transmission_username,omitempty"`
	TransmissionPassword  string `json:"transmission_password,omitempty"`
	TMDBReadAccessToken   string `json:"tmdb_read_access_token,omitempty"`
	TMDBLanguage          string `json:"tmdb_language"`
	TMDBCachePath         string `json:"tmdb_cache_path"`
	TMDBEnrichUnmapped    bool   `json:"tmdb_enrich_unmapped"`
	PoiskKinoAPIToken     string `json:"poiskkino_api_token,omitempty"`
	CatalogPath           string `json:"catalog_path"`
	MediaRoot             string `json:"media_root"`
	SMBPrefix             string `json:"smb_prefix"`
	RequestTimeoutSeconds int    `json:"request_timeout_seconds"`
}

func Load(filename string) (Config, error) {
	cfg := Config{
		CatalogPath: "web/catalog.json", TMDBLanguage: "en-US",
		TMDBCachePath: ".cache/tmdb.json", RequestTimeoutSeconds: 15,
	}
	if filename != "" {
		data, err := os.ReadFile(filename)
		if err != nil {
			return Config{}, fmt.Errorf("read config %q: %w", filename, err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse config %q: %w", filename, err)
		}
	}
	applyEnv := func(name string, target *string) {
		if value, ok := os.LookupEnv(name); ok {
			*target = value
		}
	}
	applyEnv("NAVI_TRANSMISSION_RPC", &cfg.TransmissionRPC)
	applyEnv("NAVI_TRANSMISSION_USERNAME", &cfg.TransmissionUsername)
	applyEnv("NAVI_TRANSMISSION_PASSWORD", &cfg.TransmissionPassword)
	applyEnv("NAVI_TMDB_READ_ACCESS_TOKEN", &cfg.TMDBReadAccessToken)
	applyEnv("NAVI_TMDB_LANGUAGE", &cfg.TMDBLanguage)
	applyEnv("NAVI_TMDB_CACHE_PATH", &cfg.TMDBCachePath)
	applyEnv("NAVI_POISKINO_API_TOKEN", &cfg.PoiskKinoAPIToken)
	applyEnv("NAVI_CATALOG_PATH", &cfg.CatalogPath)
	applyEnv("NAVI_MEDIA_ROOT", &cfg.MediaRoot)
	applyEnv("NAVI_SMB_PREFIX", &cfg.SMBPrefix)
	if value, ok := os.LookupEnv("NAVI_TMDB_ENRICH_UNMAPPED"); ok {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse NAVI_TMDB_ENRICH_UNMAPPED: %w", err)
		}
		cfg.TMDBEnrichUnmapped = enabled
	}
	if value, ok := os.LookupEnv("NAVI_REQUEST_TIMEOUT_SECONDS"); ok {
		seconds, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse NAVI_REQUEST_TIMEOUT_SECONDS: %w", err)
		}
		cfg.RequestTimeoutSeconds = seconds
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.TransmissionRPC) == "" {
		return errors.New("transmission_rpc is required")
	}
	u, err := url.Parse(c.TransmissionRPC)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("transmission_rpc must be an absolute HTTP(S) URL")
	}
	if strings.TrimSpace(c.CatalogPath) == "" {
		return errors.New("catalog_path is required")
	}
	if strings.TrimSpace(c.MediaRoot) == "" || !filepath.IsAbs(c.MediaRoot) {
		return errors.New("media_root must be an absolute filesystem path")
	}
	if !strings.HasPrefix(strings.ToLower(c.SMBPrefix), "smb://") {
		return errors.New("smb_prefix must begin with smb://")
	}
	if strings.TrimSpace(c.TMDBReadAccessToken) != "" {
		if strings.TrimSpace(c.TMDBLanguage) == "" {
			return errors.New("tmdb_language is required when a TMDB token is configured")
		}
		if strings.TrimSpace(c.TMDBCachePath) == "" {
			return errors.New("tmdb_cache_path is required when a TMDB token is configured")
		}
	}
	if strings.TrimSpace(c.PoiskKinoAPIToken) != "" && strings.TrimSpace(c.TMDBReadAccessToken) == "" {
		return errors.New("poiskkino_api_token requires tmdb_read_access_token")
	}
	if c.RequestTimeoutSeconds < 0 {
		return errors.New("request_timeout_seconds cannot be negative")
	}
	return nil
}

func (c Config) MetadataCacheProfile() string {
	profile := c.TMDBLanguage
	if strings.TrimSpace(c.PoiskKinoAPIToken) != "" {
		profile += "+poiskkino"
	}
	return profile
}

func (c Config) RequestTimeout() time.Duration {
	if c.RequestTimeoutSeconds == 0 {
		return 15 * time.Second
	}
	return time.Duration(c.RequestTimeoutSeconds) * time.Second
}
