package parser

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	yearPattern = regexp.MustCompile(`(?:^|[ ._\-(])((?:19|20)\d{2})(?:$|[ ._)\]-])`)
	tagPattern  = regexp.MustCompile(`(?i)\b(?:2160p|1080p|720p|480p|4k|uhd|bluray|blu-ray|bdrip|brrip|web[ ._-]?dl|webrip|hdrip|dvdrip|remux|x26[45]|h[ ._-]?26[45]|hevc|av1|hdr10\+?|hdr|dolby[ ._-]?vision|dovi|aac(?:2\.0)?|ac3|eac3|ddp?\d(?:\.\d)?|dts(?:-hd)?|truehd|atmos|multi|proper|repack|extended|unrated)\b.*$`)
	spaces      = regexp.MustCompile(`\s+`)
)

// Title extracts a human-readable title and likely release year from a torrent name.
func Title(name string) (string, int) {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	year := 0
	if match := yearPattern.FindStringSubmatch(base); len(match) > 1 {
		year, _ = strconv.Atoi(match[1])
		base = base[:yearPattern.FindStringIndex(base)[0]]
	}
	base = strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(base)
	base = tagPattern.ReplaceAllString(base, "")
	base = spaces.ReplaceAllString(strings.TrimSpace(base), " ")
	if base == "" {
		base = strings.TrimSpace(name)
	}
	return titleCase(base), year
}

func titleCase(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		if word == strings.ToUpper(word) && len(word) > 1 {
			continue
		}
		runes := []rune(strings.ToLower(word))
		if len(runes) > 0 {
			runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
