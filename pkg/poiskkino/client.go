package poiskkino

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"
)

const defaultBaseURL = "https://api.poiskkino.dev"

var ErrNotFound = errors.New("movie not found on PoiskKino")

type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("PoiskKino returned %s: %s", e.Status, e.Body)
}

func IsAuthenticationError(err error) bool {
	var httpError *HTTPError
	return errors.As(err, &httpError) && (httpError.StatusCode == http.StatusUnauthorized || httpError.StatusCode == http.StatusForbidden)
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type Match struct {
	ID     int
	TMDBID int
	Name   string
	Year   int
}

type searchResponse struct {
	Docs []searchResult `json:"docs"`
}

type searchResult struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	AlternativeName string `json:"alternativeName"`
	EnglishName     string `json:"enName"`
	Year            int    `json:"year"`
	Names           []struct {
		Name string `json:"name"`
	} `json:"names"`
	ExternalID struct {
		TMDB int `json:"tmdb"`
	} `json:"externalId"`
}

func New(token string, httpClient *http.Client) *Client {
	return NewWithBaseURL(defaultBaseURL, token, httpClient)
}

func NewWithBaseURL(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: strings.TrimSpace(token), http: httpClient}
}

// Resolve searches PoiskKino only as an identifier resolver. The returned
// TMDB ID lets the caller continue using TMDB as the canonical metadata store.
func (c *Client) Resolve(ctx context.Context, candidates []string, year int) (Match, error) {
	best, bestScore := Match{}, -1<<30
	minimum := 90
	if year > 0 {
		minimum = 140
	}
	for _, candidate := range candidates {
		query := url.Values{"query": {candidate}, "page": {"1"}, "limit": {"10"}}
		var search searchResponse
		if err := c.get(ctx, "/v1.5/movie/search", query, &search); err != nil {
			return Match{}, err
		}
		for index, result := range search.Docs {
			if result.ExternalID.TMDB <= 0 {
				continue
			}
			score := matchScore(result, candidate, year) - index
			if score > bestScore {
				bestScore = score
				best = Match{ID: result.ID, TMDBID: result.ExternalID.TMDB, Name: result.Name, Year: result.Year}
			}
		}
		if bestScore >= minimum {
			break
		}
	}
	if best.TMDBID <= 0 || bestScore < minimum {
		return Match{}, ErrNotFound
	}
	return best, nil
}

func (c *Client) get(ctx context.Context, endpoint string, query url.Values, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return fmt.Errorf("create PoiskKino request: %w", err)
	}
	request.Header.Set("X-API-KEY", c.token)
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("call PoiskKino: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return &HTTPError{StatusCode: response.StatusCode, Status: response.Status, Body: strings.TrimSpace(string(body))}
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode PoiskKino response: %w", err)
	}
	return nil
}

func matchScore(result searchResult, title string, year int) int {
	wanted := normalize(title)
	score := 0
	names := []string{result.Name, result.AlternativeName, result.EnglishName}
	for _, name := range result.Names {
		names = append(names, name.Name)
	}
	for _, name := range names {
		if normalize(name) == wanted {
			score += 100
			break
		}
	}
	if year > 0 && result.Year == year {
		score += 150
	} else if year > 0 && result.Year > 0 {
		delta := result.Year - year
		if delta < 0 {
			delta = -delta
		}
		score -= 100 + delta*5
	}
	return score
}

func normalize(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}
