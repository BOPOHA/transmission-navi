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
