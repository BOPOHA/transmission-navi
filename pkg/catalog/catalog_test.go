package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BOPOHA/transmission-navi/pkg/transmission"
)

func TestBuildMapsNASPathToSMB(t *testing.T) {
	data := Build([]transmission.Torrent{{
		ID: 1, Name: "The.Addams.Family.1991.1080p.mkv", DownloadDir: "/media/library",
		IsFinished: true, PercentDone: 1, Status: 6,
		Files: []transmission.File{{Name: "The Addams Family/movie.mkv", Length: 42, BytesCompleted: 42}},
	}}, "/media/library", "smb://example-nas/media/", time.Unix(10, 0))
	if got := data.Movies[0].PrimaryPlayURL; got != "smb://example-nas/media/The%20Addams%20Family/movie.mkv" {
		t.Fatalf("unexpected play URL: %s", got)
	}
	if data.Movies[0].Title != "The Addams Family" || data.Movies[0].Year != 1991 {
		t.Fatalf("unexpected parsed metadata: %#v", data.Movies[0])
	}
}

func TestWriteAtomic(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "nested", "catalog.json")
	want := Catalog{Version: 123, Movies: []Movie{}}
	if err := WriteAtomic(filename, want); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	var got Catalog
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Version != want.Version {
		t.Fatalf("got version %d", got.Version)
	}
}
