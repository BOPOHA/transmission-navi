package tmdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/search/movie":
			if r.URL.Query().Get("year") != "1991" || r.URL.Query().Get("query") != "The Addams Family" {
				t.Errorf("unexpected search query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"results":[{"id":1,"title":"Wrong","release_date":"2020-01-01"},{"id":2907,"title":"The Addams Family","original_title":"The Addams Family","release_date":"1991-11-22"}]}`))
		case r.URL.Path == "/movie/2907":
			_, _ = w.Write([]byte(`{"id":2907,"title":"The Addams Family","original_title":"The Addams Family","overview":"Test overview","release_date":"1991-11-22","vote_average":7.1,"poster_path":"/poster.jpg","genres":[{"name":"Comedy"},{"name":"Fantasy"}],"release_dates":{"results":[{"iso_3166_1":"US","release_dates":[{"certification":"R"}]}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	metadata, err := NewWithBaseURL(server.URL, "test-token", "en-US", server.Client()).Lookup(context.Background(), "The Addams Family", 1991)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ID != 2907 || metadata.Year != 1991 || metadata.MinAge == nil || *metadata.MinAge != 16 {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	if !strings.HasSuffix(metadata.PosterURL, "/poster.jpg") || len(metadata.Genres) != 2 {
		t.Fatalf("missing metadata fields: %#v", metadata)
	}
}

func TestLookupNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()
	_, err := NewWithBaseURL(server.URL, "token", "en-US", server.Client()).Lookup(context.Background(), "Missing", 2020)
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestLookupRetriesRussianTransliteration(t *testing.T) {
	queries := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/search/movie":
			query := r.URL.Query().Get("query")
			queries = append(queries, query)
			if query == "Немыслимое" {
				_, _ = w.Write([]byte(`{"results":[{"id":43539,"title":"Unthinkable","original_title":"Unthinkable","release_date":"2010-06-14"}]}`))
			} else {
				_, _ = w.Write([]byte(`{"results":[]}`))
			}
		case "/movie/43539":
			_, _ = w.Write([]byte(`{"id":43539,"title":"Unthinkable","original_title":"Unthinkable","release_date":"2010-06-14","genres":[],"release_dates":{"results":[]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	metadata, err := NewWithBaseURL(server.URL, "token", "en-US", server.Client()).Lookup(context.Background(), "Nemyslimoe", 2010)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ID != 43539 || !reflect.DeepEqual(queries, []string{"Nemyslimoe", "Немыслимое"}) {
		t.Fatalf("unexpected fallback: metadata=%#v queries=%#v", metadata, queries)
	}
}
