package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

const (
	defaultBaseURL  = "https://api.themoviedb.org/3"
	defaultImageURL = "https://image.tmdb.org/t/p/w500"
)

var ErrNotFound = errors.New("movie not found on TMDB")

type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("TMDB returned %s: %s", e.Status, e.Body)
}

func IsAuthenticationError(err error) bool {
	var httpError *HTTPError
	return errors.As(err, &httpError) && (httpError.StatusCode == http.StatusUnauthorized || httpError.StatusCode == http.StatusForbidden)
}

type Client struct {
	baseURL  string
	token    string
	language string
	http     *http.Client
}

type Metadata struct {
	ID            int      `json:"id"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title,omitempty"`
	Overview      string   `json:"overview,omitempty"`
	Year          int      `json:"year,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Rating        float64  `json:"rating"`
	PosterURL     string   `json:"poster_url,omitempty"`
	MinAge        *int     `json:"min_age,omitempty"`
	Certification string   `json:"certification,omitempty"`
}

type searchResult struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	ReleaseDate   string  `json:"release_date"`
	Popularity    float64 `json:"popularity"`
}

type searchResponse struct {
	Results []searchResult `json:"results"`
}

type detailsResponse struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	ReleaseDate   string  `json:"release_date"`
	VoteAverage   float64 `json:"vote_average"`
	PosterPath    string  `json:"poster_path"`
	Genres        []struct {
		Name string `json:"name"`
	} `json:"genres"`
	ReleaseDates struct {
		Results []struct {
			Country string `json:"iso_3166_1"`
			Dates   []struct {
				Certification string `json:"certification"`
			} `json:"release_dates"`
		} `json:"results"`
	} `json:"release_dates"`
}

func New(token, language string, httpClient *http.Client) *Client {
	return NewWithBaseURL(defaultBaseURL, token, language, httpClient)
}

func NewWithBaseURL(baseURL, token, language string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if language == "" {
		language = "en-US"
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: strings.TrimSpace(token), language: language, http: httpClient}
}

func (c *Client) Lookup(ctx context.Context, title string, year int) (Metadata, error) {
	query := url.Values{"query": {title}, "language": {c.language}}
	if year > 0 {
		query.Set("year", strconv.Itoa(year))
	}
	var search searchResponse
	if err := c.get(ctx, "/search/movie", query, &search); err != nil {
		return Metadata{}, err
	}
	if len(search.Results) == 0 {
		return Metadata{}, ErrNotFound
	}
	match := bestMatch(search.Results, title, year)

	query = url.Values{"language": {c.language}, "append_to_response": {"release_dates"}}
	var details detailsResponse
	if err := c.get(ctx, "/movie/"+strconv.Itoa(match.ID), query, &details); err != nil {
		return Metadata{}, err
	}
	metadata := Metadata{
		ID: details.ID, Title: details.Title, OriginalTitle: details.OriginalTitle,
		Overview: details.Overview, Year: releaseYear(details.ReleaseDate),
		Rating: details.VoteAverage, Genres: make([]string, 0, len(details.Genres)),
	}
	for _, genre := range details.Genres {
		metadata.Genres = append(metadata.Genres, genre.Name)
	}
	if details.PosterPath != "" {
		metadata.PosterURL = defaultImageURL + details.PosterPath
	}
	metadata.Certification, metadata.MinAge = certification(details)
	return metadata, nil
}

func (c *Client) get(ctx context.Context, endpoint string, query url.Values, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return fmt.Errorf("create TMDB request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("call TMDB: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return &HTTPError{StatusCode: response.StatusCode, Status: response.Status, Body: strings.TrimSpace(string(body))}
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode TMDB response: %w", err)
	}
	return nil
}

func bestMatch(results []searchResult, title string, year int) searchResult {
	wanted := normalize(title)
	best, bestScore := results[0], -1<<30
	for index, result := range results {
		score := -index
		if normalize(result.Title) == wanted || normalize(result.OriginalTitle) == wanted {
			score += 100
		}
		resultYear := releaseYear(result.ReleaseDate)
		if year > 0 && resultYear == year {
			score += 50
		} else if year > 0 && resultYear > 0 {
			delta := resultYear - year
			if delta < 0 {
				delta = -delta
			}
			score -= delta * 5
		}
		if score > bestScore {
			best, bestScore = result, score
		}
	}
	return best
}

func normalize(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}

func releaseYear(date string) int {
	if len(date) < 4 {
		return 0
	}
	year, _ := strconv.Atoi(date[:4])
	return year
}

func certification(details detailsResponse) (string, *int) {
	priorities := []string{"US", "GB", "DE", "FR", "RU"}
	for _, country := range priorities {
		bestAge := -1
		bestLabel := ""
		for _, result := range details.ReleaseDates.Results {
			if result.Country != country {
				continue
			}
			for _, release := range result.Dates {
				if age, ok := normalizedAge(release.Certification); ok && age > bestAge {
					bestAge, bestLabel = age, release.Certification
				}
			}
		}
		if bestAge >= 0 {
			age := bestAge
			return bestLabel, &age
		}
	}
	return "", nil
}

func normalizedAge(value string) (int, bool) {
	value = strings.ToUpper(strings.TrimSpace(value))
	ages := map[string]int{
		"G": 0, "U": 0, "A": 0, "AL": 0, "TP": 0,
		"PG": 6, "6": 6, "7": 6, "TV-Y7": 6,
		"PG-13": 12, "12": 12, "12A": 12, "13": 12, "TV-14": 12,
		"R": 16, "15": 16, "16": 16, "FSK 16": 16,
		"NC-17": 18, "18": 18, "FSK 18": 18, "TV-MA": 18,
	}
	age, ok := ages[value]
	return age, ok
}
