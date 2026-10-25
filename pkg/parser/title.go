package parser

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	yearPattern         = regexp.MustCompile(`(?:^|[ ._\-(])((?:19|20)\d{2})(?:$|[ ._)\]-])`)
	tagPattern          = regexp.MustCompile(`(?i)\b(?:2160p|1080p|720p|480p|4k|uhd|bluray|blu-ray|bdrip|brrip|web[ ._-]?dl|webrip|hdrip|dvdrip|remux|x26[45]|h[ ._-]?26[45]|hevc|av1|hdr10\+?|hdr|dolby[ ._-]?vision|dovi|aac(?:2\.0)?|ac3|eac3|ddp?\d(?:\.\d)?|dts(?:-hd)?|truehd|atmos|multi|proper|repack|extended|unrated)\b.*$`)
	releaseGroupPattern = compileReleaseGroupPattern()
	spaces              = regexp.MustCompile(`\s+`)
	nonAlphaNum         = regexp.MustCompile(`[^A-Z0-9]+`)
)

// releaseGroups contains publisher/uploader labels that commonly appear at
// the end of torrent names but are not part of a movie or series title.
var releaseGroups = []string{
	"LostFilm", "teko", "ExKinoRay", "GeneralFilm", "EniaHD", "Jaskier",
	"RGzsRutracker", "Dalemake", "Nicodem", "Files-x", "ylnian", "il68k",
	"ad3wal3", "Frost O.S", "kontarik81", "Aleksan55", "HQ-HOME",
}

var threeDTags = map[string]bool{
	"3D": true, "SBS": true, "HSBS": true, "FSBS": true,
	"OU": true, "HOU": true, "FOU": true, "TAB": true, "MVC": true,
}

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
	for releaseGroupPattern.MatchString(base) {
		base = releaseGroupPattern.ReplaceAllString(base, "")
	}
	base = spaces.ReplaceAllString(strings.TrimSpace(base), " ")
	if base == "" {
		base = strings.TrimSpace(name)
	}
	return titleCase(base), year
}

func compileReleaseGroupPattern() *regexp.Regexp {
	groups := make([]string, 0, len(releaseGroups))
	separator := regexp.MustCompile(`[._-]+`)
	for _, group := range releaseGroups {
		normalized := separator.ReplaceAllString(group, " ")
		words := strings.Fields(normalized)
		for index, word := range words {
			words[index] = regexp.QuoteMeta(word)
		}
		groups = append(groups, strings.Join(words, `\s+`))
	}
	return regexp.MustCompile(`(?i)\s+(?:[\[(]\s*)?(?:(?:rip\s+)?by\s+)?(?:` + strings.Join(groups, "|") + `)\s*[\])]?\s*$`)
}

// Is3D recognizes the common stereoscopic tags used in torrent and media
// filenames. Values should include both the torrent name and its file names so
// releases whose 3D marker appears in only one of those places are detected.
func Is3D(values ...string) bool {
	for _, value := range values {
		normalized := nonAlphaNum.ReplaceAllString(strings.ToUpper(value), " ")
		fields := strings.Fields(normalized)
		for _, field := range fields {
			if threeDTags[field] {
				return true
			}
		}
		phrase := " " + strings.Join(fields, " ") + " "
		if strings.Contains(phrase, " SIDE BY SIDE ") ||
			strings.Contains(phrase, " TOP BOTTOM ") ||
			strings.Contains(phrase, " OVER UNDER ") {
			return true
		}
	}
	return false
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
