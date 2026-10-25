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
	if data.Movies[0].VideoMode != "2d" {
		t.Fatalf("unexpected video mode: %s", data.Movies[0].VideoMode)
	}
}

func TestBuildDetects3DFromMediaFile(t *testing.T) {
	data := Build([]transmission.Torrent{{
		ID: 2, Name: "Avatar 2009", DownloadDir: "/media/library",
		Files: []transmission.File{{Name: "Avatar.2009.HSBS.3D.mkv", Length: 42}},
	}}, "/media/library", "smb://example-nas/media", time.Unix(10, 0))
	if data.Movies[0].VideoMode != "3d" {
		t.Fatalf("unexpected video mode: %s", data.Movies[0].VideoMode)
	}
}

func TestBuildSplitsMultiMovieTorrent(t *testing.T) {
	data := Build([]transmission.Torrent{{
		ID: 3, Name: "В поисках Нэмо-Дори", DownloadDir: "/media/library", Status: 4,
		Files: []transmission.File{
			{Name: "В поисках Нэмо-Дори/В поисках Дори.hdr.mp4", Length: 3900, BytesCompleted: 1950},
			{Name: "В поисках Нэмо-Дори/В поисках Немо.hdr.mp4", Length: 4000, BytesCompleted: 4000},
		},
	}}, "/media/library", "smb://example-nas/media", time.Unix(10, 0))

	if len(data.Movies) != 2 {
		t.Fatalf("got %d catalog entries, want 2", len(data.Movies))
	}
	if data.Movies[0].Title != "В Поисках Дори" || data.Movies[1].Title != "В Поисках Немо" {
		t.Fatalf("unexpected titles: %q, %q", data.Movies[0].Title, data.Movies[1].Title)
	}
	if len(data.Movies[0].Files) != 1 || len(data.Movies[1].Files) != 1 {
		t.Fatalf("split entries should each contain one movie file: %#v", data.Movies)
	}
	if data.Movies[0].PercentDone != 50 || data.Movies[0].Finished {
		t.Fatalf("unexpected per-file progress: %#v", data.Movies[0])
	}
	if data.Movies[1].PercentDone != 100 || !data.Movies[1].Finished {
		t.Fatalf("unexpected per-file completion: %#v", data.Movies[1])
	}
}

func TestBuildSplitsMultiMovieTorrentWithIndividualYears(t *testing.T) {
	data := Build([]transmission.Torrent{{
		ID: 4, Name: "Semejka.Addams.(2.filma).1991.1993.DivX.DVDRip", DownloadDir: "/media/library",
		Files: []transmission.File{
			{Name: "Semejka.Addams.(2.filma).1991.1993.DivX.DVDRip/Semejka.Addams.1.1991.DivX.DVDRip.avi", Length: 1500},
			{Name: "Semejka.Addams.(2.filma).1991.1993.DivX.DVDRip/Semejka.Addams.2.1993.DivX.DVDRip.avi", Length: 700},
		},
	}}, "/media/library", "smb://example-nas/media", time.Unix(10, 0))

	if len(data.Movies) != 2 {
		t.Fatalf("got %d catalog entries, want 2", len(data.Movies))
	}
	if data.Movies[0].Title != "Semejka Addams 1" || data.Movies[0].Year != 1991 {
		t.Fatalf("unexpected first movie: %#v", data.Movies[0])
	}
	if data.Movies[1].Title != "Semejka Addams 2" || data.Movies[1].Year != 1993 {
		t.Fatalf("unexpected second movie: %#v", data.Movies[1])
	}
}

func TestBuildKeepsSeasonAsOneEntry(t *testing.T) {
	data := Build([]transmission.Torrent{{
		ID: 5, Name: "Example.Show.S01.1080p", DownloadDir: "/media/library",
		Files: []transmission.File{
			{Name: "Example.Show.S01E01.mkv", Length: 1000},
			{Name: "Example.Show.S01E02.mkv", Length: 1000},
		},
	}}, "/media/library", "smb://example-nas/media", time.Unix(10, 0))

	if len(data.Movies) != 1 || len(data.Movies[0].Files) != 2 {
		t.Fatalf("season should remain one catalog entry: %#v", data.Movies)
	}
}

func TestBuildKeepsNumberedSeriesAsOneEntry(t *testing.T) {
	data := Build([]transmission.Torrent{{
		ID: 7, Name: "Dinozavr.WEBRip", DownloadDir: "/media/library",
		Files: []transmission.File{
			{Name: "Dinozavr/01.Dinozavr.WEBRip.mkv", Length: 1000},
			{Name: "Dinozavr/02.Dinozavr.WEBRip.mkv", Length: 1000},
			{Name: "Dinozavr/03.Dinozavr.WEBRip.mkv", Length: 1000},
		},
	}}, "/media/library", "smb://example-nas/media", time.Unix(10, 0))

	if len(data.Movies) != 1 || len(data.Movies[0].Files) != 3 {
		t.Fatalf("numbered series should remain one catalog entry: %#v", data.Movies)
	}
}

func TestBuildStillSplitsNumberedMovieCollection(t *testing.T) {
	data := Build([]transmission.Torrent{{
		ID: 8, Name: "БЕТХОВЕН", DownloadDir: "/media/library",
		Files: []transmission.File{
			{Name: "БЕТХОВЕН/Бетховен-1.avi", Length: 1000},
			{Name: "БЕТХОВЕН/Бетховен-2.avi", Length: 1000},
			{Name: "БЕТХОВЕН/Бетховен-3.avi", Length: 1000},
		},
	}}, "/media/library", "smb://example-nas/media", time.Unix(10, 0))

	if len(data.Movies) != 3 {
		t.Fatalf("movie collection should be split, got %#v", data.Movies)
	}
}

func TestBuildIgnoresTrailerVariantsWhenLookingForMoviePack(t *testing.T) {
	data := Build([]transmission.Torrent{{
		ID: 9, Name: "Avatar Fire and Ash 2025", DownloadDir: "/media/library",
		Files: []transmission.File{
			{Name: "AvatarFireAsh_IMAX_TLR-1_3840x2076.mkv", Length: 1000},
			{Name: "AvatarFireAsh_IMAX_TLR-2_3D-SBS_3840x1038.mkv", Length: 1000},
		},
	}}, "/media/library", "smb://example-nas/media", time.Unix(10, 0))

	if len(data.Movies) != 1 || len(data.Movies[0].Files) != 2 {
		t.Fatalf("trailer variants should remain one catalog entry: %#v", data.Movies)
	}
}

func TestBuildGroupsAlternateEncodesOfSameMovie(t *testing.T) {
	data := Build([]transmission.Torrent{{
		ID: 6, Name: "Two movie pack", DownloadDir: "/media/library",
		Files: []transmission.File{
			{Name: "Pack/First Movie 1991 (1080p HD).mkv", Length: 3000},
			{Name: "Pack/First Movie 1991 (HD).mkv", Length: 2000},
			{Name: "Pack/Second Movie 1992.mkv", Length: 2500},
		},
	}}, "/media/library", "smb://example-nas/media", time.Unix(10, 0))

	if len(data.Movies) != 2 {
		t.Fatalf("got %d catalog entries, want 2: %#v", len(data.Movies), data.Movies)
	}
	if data.Movies[0].Title != "First Movie" || len(data.Movies[0].Files) != 2 {
		t.Fatalf("alternate encodes were not grouped: %#v", data.Movies[0])
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
