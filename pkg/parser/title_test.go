package parser

import "testing"

func TestTitle(t *testing.T) {
	tests := []struct {
		input string
		title string
		year  int
	}{
		{"The.Addams.Family.1991.1080p.BluRay.x264.mkv", "The Addams Family", 1991},
		{"Dune Part Two (2024) 2160p WEB-DL", "Dune Part Two", 2024},
		{"MY_MOVIE_1080p_WEBRip", "MY MOVIE", 0},
	}
	for _, test := range tests {
		title, year := Title(test.input)
		if title != test.title || year != test.year {
			t.Errorf("Title(%q) = %q, %d; want %q, %d", test.input, title, year, test.title, test.year)
		}
	}
}

func TestIs3D(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Avatar.2009.3D.HSBS.1080p.mkv", true},
		{"Dune 3D Half-OU BluRay", true},
		{"movie/MVC/main.mkv", true},
		{"Nature Side by Side 2025.mp4", true},
		{"Apollo 13 1995 1080p.mkv", false},
		{"Three Days 2024 3Disc BluRay", false},
	}
	for _, test := range tests {
		if got := Is3D(test.name); got != test.want {
			t.Errorf("Is3D(%q) = %v; want %v", test.name, got, test.want)
		}
	}
}
