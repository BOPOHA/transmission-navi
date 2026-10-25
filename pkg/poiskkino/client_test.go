package poiskkino

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveTMDBID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "test-token" {
			t.Errorf("missing API key")
		}
		if r.URL.Path != "/v1.5/movie/search" || r.URL.Query().Get("limit") != "10" {
			t.Errorf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"docs":[{"id":1,"name":"Семейка Аддамс","alternativeName":"The Addams Family","year":1991,"externalId":{"tmdb":2907}}]}`))
	}))
	defer server.Close()

	match, err := NewWithBaseURL(server.URL, "test-token", server.Client()).Resolve(context.Background(), []string{"Semejka Addams 1", "Семейка Аддамс"}, 1991)
	if err != nil {
		t.Fatal(err)
	}
	if match.ID != 1 || match.TMDBID != 2907 || match.Year != 1991 {
		t.Fatalf("unexpected match: %#v", match)
	}
}

func TestResolveRequiresStrongMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"docs":[{"id":1,"name":"Different Movie","year":2001,"externalId":{"tmdb":42}}]}`))
	}))
	defer server.Close()

	_, err := NewWithBaseURL(server.URL, "token", server.Client()).Resolve(context.Background(), []string{"Missing Movie"}, 2020)
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestResolvePrefersReleaseYearOverWrongExactTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"docs":[{"id":1,"name":"Semejka Addams 2","year":1991,"externalId":{"tmdb":2907}},{"id":2,"name":"Ценности семейки Аддамс","year":1993,"externalId":{"tmdb":2758}}]}`))
	}))
	defer server.Close()

	match, err := NewWithBaseURL(server.URL, "token", server.Client()).Resolve(context.Background(), []string{"Semejka Addams 2"}, 1993)
	if err != nil {
		t.Fatal(err)
	}
	if match.ID != 2 || match.TMDBID != 2758 {
		t.Fatalf("wrong release selected: %#v", match)
	}
}

func TestResolveIgnoresResultWithoutTMDBID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"docs":[{"id":1,"name":"Movie","year":2020,"externalId":{"imdb":"tt1"}}]}`))
	}))
	defer server.Close()

	_, err := NewWithBaseURL(server.URL, "token", server.Client()).Resolve(context.Background(), []string{"Movie"}, 2020)
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}
