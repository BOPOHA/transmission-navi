package transmission

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTorrentsHandlesSessionHandshake(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get(sessionHeader) != "test-session" {
			w.Header().Set(sessionHeader, "test-session")
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"arguments":{"torrents":[{"id":7,"name":"Movie.2024.1080p.mkv","files":[]}]},"result":"success"}`))
	}))
	defer server.Close()

	torrents, err := New(server.URL, "", "", server.Client()).Torrents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("got %d requests, want handshake plus retry", requests)
	}
	if len(torrents) != 1 || torrents[0].ID != 7 {
		t.Fatalf("unexpected torrents: %#v", torrents)
	}
}
