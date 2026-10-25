package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/BOPOHA/transmission-navi/internal/buildinfo"
	"github.com/BOPOHA/transmission-navi/pkg/catalog"
	"github.com/BOPOHA/transmission-navi/pkg/config"
	"github.com/BOPOHA/transmission-navi/pkg/tmdb"
	"github.com/BOPOHA/transmission-navi/pkg/transmission"
)

func main() {
	configPath := flag.String("config", "", "optional path to a JSON configuration file (environment variables override it)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(buildinfo.Version)
		return
	}
	log.SetFlags(0)
	if err := run(*configPath); err != nil {
		log.Printf("navi-indexer: %v", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	httpClient := &http.Client{Timeout: cfg.RequestTimeout()}
	client := transmission.New(cfg.TransmissionRPC, cfg.TransmissionUsername, cfg.TransmissionPassword, httpClient)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout())
	defer cancel()
	torrents, err := client.Torrents(ctx)
	if err != nil {
		return err
	}
	output := catalog.Build(torrents, cfg.MediaRoot, cfg.SMBPrefix, time.Now())
	if cfg.TMDBReadAccessToken != "" {
		if err := enrichTMDB(&output, cfg, httpClient); err != nil {
			return err
		}
	}
	if err := catalog.WriteAtomic(cfg.CatalogPath, output); err != nil {
		return err
	}
	fmt.Printf("indexed %d torrents into %s\n", len(output.Movies), cfg.CatalogPath)
	return nil
}

func enrichTMDB(output *catalog.Catalog, cfg config.Config, httpClient *http.Client) error {
	cache, err := tmdb.LoadCache(cfg.TMDBCachePath)
	if err != nil {
		return err
	}
	client := tmdb.New(cfg.TMDBReadAccessToken, cfg.TMDBLanguage, httpClient)
	now := time.Now()
	enriched, cacheHits, notFound, failures := 0, 0, 0, 0
	for index := range output.Movies {
		movie := &output.Movies[index]
		if !cfg.TMDBEnrichUnmapped && movie.PrimaryPlayURL == "" {
			movie.TMDBStatus = "not_checked"
			continue
		}
		entry, cached := cache.Get(movie.Title, movie.Year, cfg.TMDBLanguage, now)
		if cached {
			cacheHits++
			if entry.Found {
				applyTMDB(movie, entry.Metadata)
				movie.TMDBStatus = "matched"
				enriched++
			} else {
				movie.TMDBStatus = "not_found"
				notFound++
			}
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout())
		metadata, lookupErr := client.Lookup(ctx, movie.Title, movie.Year)
		cancel()
		if lookupErr != nil {
			if tmdb.IsAuthenticationError(lookupErr) {
				return fmt.Errorf("authenticate with TMDB: %w", lookupErr)
			}
			if lookupErr == tmdb.ErrNotFound {
				cache.Put(movie.Title, movie.Year, cfg.TMDBLanguage, tmdb.Metadata{}, false, now)
				movie.TMDBStatus = "not_found"
				notFound++
				continue
			}
			failures++
			movie.TMDBStatus = "error"
			log.Printf("TMDB lookup failed for %q: %v", movie.Title, lookupErr)
			continue
		}
		cache.Put(movie.Title, movie.Year, cfg.TMDBLanguage, metadata, true, now)
		applyTMDB(movie, metadata)
		movie.TMDBStatus = "matched"
		enriched++
	}
	if err := cache.WriteAtomic(cfg.TMDBCachePath); err != nil {
		return err
	}
	fmt.Printf("TMDB: enriched %d movies (%d cache hits, %d not found, %d transient failures)\n", enriched, cacheHits, notFound, failures)
	return nil
}

func applyTMDB(movie *catalog.Movie, metadata tmdb.Metadata) {
	movie.TMDBID = metadata.ID
	if metadata.Title != "" {
		movie.Title = metadata.Title
	}
	movie.OriginalTitle = metadata.OriginalTitle
	if metadata.Year > 0 {
		movie.Year = metadata.Year
	}
	movie.Overview = metadata.Overview
	movie.Genres = metadata.Genres
	movie.RatingTMDB = metadata.Rating
	movie.MinAge = metadata.MinAge
	movie.Certification = metadata.Certification
	movie.PosterURL = metadata.PosterURL
}
