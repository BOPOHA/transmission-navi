package catalog

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BOPOHA/transmission-navi/pkg/parser"
	"github.com/BOPOHA/transmission-navi/pkg/transmission"
)

var videoExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".avi": true, ".mov": true,
	".m4v": true, ".webm": true, ".mpg": true, ".mpeg": true, ".ts": true,
}

type Catalog struct {
	Version   int64   `json:"version"`
	UpdatedAt string  `json:"updated_at"`
	Movies    []Movie `json:"movies"`
}

type Movie struct {
	ID             int64    `json:"id"`
	HashString     string   `json:"hash_string"`
	TorrentName    string   `json:"torrent_name"`
	Title          string   `json:"title"`
	OriginalTitle  string   `json:"original_title,omitempty"`
	Year           int      `json:"year,omitempty"`
	Overview       string   `json:"overview,omitempty"`
	Genres         []string `json:"genres,omitempty"`
	RatingTMDB     float64  `json:"rating_tmdb,omitempty"`
	MinAge         *int     `json:"min_age,omitempty"`
	Certification  string   `json:"certification,omitempty"`
	PosterURL      string   `json:"poster_url,omitempty"`
	TMDBID         int      `json:"tmdb_id,omitempty"`
	TMDBStatus     string   `json:"tmdb_status,omitempty"`
	VideoMode      string   `json:"video_mode"`
	DownloadDir    string   `json:"download_dir"`
	Finished       bool     `json:"finished"`
	PercentDone    int      `json:"percent_done"`
	Status         string   `json:"status"`
	AddedAt        string   `json:"added_at,omitempty"`
	PrimaryPlayURL string   `json:"primary_play_url,omitempty"`
	Files          []File   `json:"files"`
}

type File struct {
	Name           string `json:"name"`
	Size           int64  `json:"size"`
	BytesCompleted int64  `json:"bytes_completed"`
	Playable       bool   `json:"playable"`
	PlayURL        string `json:"play_url,omitempty"`
}

func Build(torrents []transmission.Torrent, mediaRoot, smbPrefix string, now time.Time) Catalog {
	movies := make([]Movie, 0, len(torrents))
	for _, torrent := range torrents {
		title, year := parser.Title(torrent.Name)
		mediaNames := make([]string, 1, len(torrent.Files)+1)
		mediaNames[0] = torrent.Name
		for _, source := range torrent.Files {
			if videoExtensions[strings.ToLower(filepath.Ext(source.Name))] {
				mediaNames = append(mediaNames, source.Name)
			}
		}
		videoMode := "2d"
		if parser.Is3D(mediaNames...) {
			videoMode = "3d"
		}
		movie := Movie{
			ID: torrent.ID, HashString: torrent.HashString, TorrentName: torrent.Name,
			Title: title, Year: year, DownloadDir: torrent.DownloadDir,
			VideoMode: videoMode,
			Finished:  torrent.IsFinished, PercentDone: percent(torrent.PercentDone),
			Status: statusName(torrent.Status), Files: make([]File, 0, len(torrent.Files)),
		}
		if torrent.AddedDate > 0 {
			movie.AddedAt = time.Unix(torrent.AddedDate, 0).UTC().Format(time.RFC3339)
		}
		for _, source := range torrent.Files {
			playable := videoExtensions[strings.ToLower(filepath.Ext(source.Name))]
			playURL := ""
			if playable {
				playURL = mapSMB(filepath.Join(torrent.DownloadDir, filepath.FromSlash(source.Name)), mediaRoot, smbPrefix)
			}
			file := File{Name: source.Name, Size: source.Length, BytesCompleted: source.BytesCompleted, Playable: playable, PlayURL: playURL}
			movie.Files = append(movie.Files, file)
			if playURL != "" && (movie.PrimaryPlayURL == "" || source.Length > primarySize(movie.Files, movie.PrimaryPlayURL)) {
				movie.PrimaryPlayURL = playURL
			}
		}
		movies = append(movies, movie)
	}
	sort.SliceStable(movies, func(i, j int) bool {
		return strings.ToLower(movies[i].Title) < strings.ToLower(movies[j].Title)
	})
	return Catalog{Version: now.UnixMilli(), UpdatedAt: now.UTC().Format(time.RFC3339), Movies: movies}
}

func WriteAtomic(filename string, data Catalog) error {
	directory := filepath.Dir(filename)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create catalog directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".catalog-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary catalog: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		temporary.Close()
		return fmt.Errorf("encode catalog: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync catalog: %w", err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return fmt.Errorf("set catalog permissions: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close catalog: %w", err)
	}
	if err := os.Rename(temporaryName, filename); err != nil {
		return fmt.Errorf("publish catalog: %w", err)
	}
	return nil
}

func mapSMB(filename, mediaRoot, smbPrefix string) string {
	relative, err := filepath.Rel(filepath.Clean(mediaRoot), filepath.Clean(filename))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.TrimRight(smbPrefix, "/") + "/" + strings.Join(parts, "/")
}

func primarySize(files []File, playURL string) int64 {
	for _, file := range files {
		if file.PlayURL == playURL {
			return file.Size
		}
	}
	return 0
}

func percent(value float64) int {
	result := int(value*100 + 0.5)
	if result < 0 {
		return 0
	}
	if result > 100 {
		return 100
	}
	return result
}

func statusName(status int) string {
	name, ok := map[int]string{0: "stopped", 1: "queued to verify", 2: "verifying", 3: "queued to download", 4: "downloading", 5: "queued to seed", 6: "seeding"}[status]
	if !ok {
		return "unknown"
	}
	return name
}
