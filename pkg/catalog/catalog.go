package catalog

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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

var (
	seasonPattern       = regexp.MustCompile(`(?i)(?:^|[ ./_\-\[(])s\d{1,2}(?:e\d{1,3})?(?:$|[ ./_\-\])])|(?:season|сезон)[ ._\-]*\d+`)
	episodePattern      = regexp.MustCompile(`(?i)(?:^|[ ./_\-\[(])(?:(?:e|ep)\d{1,3}|\d{1,2}x\d{1,3})(?:$|[ ./_\-\])])|(?:^|[ ./_\-\[(])\d{1,3}[ ._\-]*(?:сер(?:ия|ии)?|эпизод|выпуск)(?:$|[ ._\-\])])|(?:^|[ ./_\-\[(])(?:сер(?:ия|ии)?|эпизод|выпуск)(?:$|[ ._\-\])])|(?:^|[ ./_\-\[(])\d{2}[._-]\d{2,3}(?:$|[ ./_\-\])])`)
	extraPattern        = regexp.MustCompile(`(?i)(?:^|[ ._\-/\[(])(?:sample|trailer|tlr[ ._\-]*\d*|тизер|трейлер|featurette|bonus)(?:$|[ ._\-/\])])`)
	qualityPattern      = regexp.MustCompile(`(?i)[ ._\-]*(?:\(?\s*(?:2160p|1080p|720p|480p|4k|uhd|hd)\s*\)?).*$`)
	identityPunctuation = regexp.MustCompile(`[^\p{L}\p{N}]+`)
	leadingNumber       = regexp.MustCompile(`^\d{1,3}\s+(.+)$`)
	trailingNumber      = regexp.MustCompile(`^(.+?)\s+\d{1,3}$`)
)

type mediaFile struct {
	source transmission.File
	file   File
	title  string
	year   int
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
		files := make([]mediaFile, 0, len(torrent.Files))
		for _, source := range torrent.Files {
			playable := videoExtensions[strings.ToLower(filepath.Ext(source.Name))]
			playURL := ""
			if playable {
				playURL = mapSMB(filepath.Join(torrent.DownloadDir, filepath.FromSlash(source.Name)), mediaRoot, smbPrefix)
			}
			title, year := "", 0
			if playable {
				title, year = parser.Title(qualityPattern.ReplaceAllString(filepath.Base(source.Name), ""))
			}
			files = append(files, mediaFile{source: source, file: File{Name: source.Name, Size: source.Length, BytesCompleted: source.BytesCompleted, Playable: playable, PlayURL: playURL}, title: title, year: year})
		}

		groups := movieGroups(torrent.Name, files)
		if len(groups) > 1 {
			for _, group := range groups {
				movies = append(movies, buildMovie(torrent, group[0].title, group[0].year, group, true))
			}
			continue
		}

		title, year := parser.Title(torrent.Name)
		movies = append(movies, buildMovie(torrent, title, year, files, false))
	}
	sort.SliceStable(movies, func(i, j int) bool {
		return strings.ToLower(movies[i].Title) < strings.ToLower(movies[j].Title)
	})
	return Catalog{Version: now.UnixMilli(), UpdatedAt: now.UTC().Format(time.RFC3339), Movies: movies}
}

func buildMovie(torrent transmission.Torrent, title string, year int, files []mediaFile, fileProgress bool) Movie {
	mediaNames := make([]string, 0, len(files)+1)
	if !fileProgress {
		mediaNames = append(mediaNames, torrent.Name)
	}
	movie := Movie{
		ID: torrent.ID, HashString: torrent.HashString, TorrentName: torrent.Name,
		Title: title, Year: year, DownloadDir: torrent.DownloadDir,
		VideoMode: "2d", Finished: torrent.IsFinished, PercentDone: percent(torrent.PercentDone),
		Status: statusName(torrent.Status), Files: make([]File, 0, len(files)),
	}
	if torrent.AddedDate > 0 {
		movie.AddedAt = time.Unix(torrent.AddedDate, 0).UTC().Format(time.RFC3339)
	}
	var primary *mediaFile
	for index := range files {
		item := &files[index]
		movie.Files = append(movie.Files, item.file)
		if item.file.Playable {
			mediaNames = append(mediaNames, item.source.Name)
		}
		if item.file.Playable && (primary == nil || item.source.Length > primary.source.Length) {
			primary = item
		}
	}
	if parser.Is3D(mediaNames...) {
		movie.VideoMode = "3d"
	}
	if primary != nil {
		movie.PrimaryPlayURL = primary.file.PlayURL
		if fileProgress && primary.source.Length > 0 {
			movie.PercentDone = percent(float64(primary.source.BytesCompleted) / float64(primary.source.Length))
			movie.Finished = primary.source.BytesCompleted >= primary.source.Length
		}
	}
	return movie
}

// movieGroups returns one group per distinct movie in a multi-film torrent.
// Obvious seasons remain one catalog entry, and alternate encodes with the
// same parsed title and year remain available together in one playlist.
func movieGroups(torrentName string, files []mediaFile) [][]mediaFile {
	playable := make([]mediaFile, 0, len(files))
	var largest int64
	for _, item := range files {
		if item.file.Playable && item.source.Length > largest {
			largest = item.source.Length
		}
	}
	for _, item := range files {
		if !item.file.Playable || extraPattern.MatchString(item.source.Name) {
			continue
		}
		if largest > 0 && item.source.Length < largest/20 && item.source.Length < 250*1024*1024 {
			continue
		}
		playable = append(playable, item)
	}
	if len(playable) < 2 || isEpisodic(torrentName, playable) {
		return nil
	}

	groups := make([][]mediaFile, 0, len(playable))
	indexes := make(map[string]int)
	for _, item := range playable {
		key := movieIdentity(item.title, item.year)
		if index, ok := indexes[key]; ok {
			groups[index] = append(groups[index], item)
			continue
		}
		indexes[key] = len(groups)
		groups = append(groups, []mediaFile{item})
	}
	if len(groups) < 2 {
		return nil
	}
	return groups
}

func movieIdentity(title string, year int) string {
	normalized := strings.ToLower(identityPunctuation.ReplaceAllString(title, ""))
	return fmt.Sprintf("%s:%d", normalized, year)
}

func isEpisodic(torrentName string, files []mediaFile) bool {
	if seasonPattern.MatchString(torrentName) || episodePattern.MatchString(torrentName) {
		return true
	}
	for _, item := range files {
		if seasonPattern.MatchString(item.source.Name) || episodePattern.MatchString(item.source.Name) {
			return true
		}
	}
	return isNumberedSeries(files)
}

func isNumberedSeries(files []mediaFile) bool {
	if len(files) < 3 {
		return false
	}
	leadingBase := ""
	allLeading := true
	allTrailing := true
	for _, item := range files {
		title := strings.ToLower(strings.TrimSpace(item.title))
		match := leadingNumber.FindStringSubmatch(title)
		if len(match) != 2 {
			allLeading = false
		} else if leadingBase == "" {
			leadingBase = match[1]
		} else if leadingBase != match[1] {
			allLeading = false
		}
		if !trailingNumber.MatchString(title) {
			allTrailing = false
		}
	}
	return allLeading || (len(files) >= 8 && allTrailing)
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
