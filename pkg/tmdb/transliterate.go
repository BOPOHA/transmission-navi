package tmdb

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	filmCountPattern = regexp.MustCompile(`(?i)\s*[\[(]\s*\d+\s+filma?\s*[\])]\s*`)
	latinWordPattern = regexp.MustCompile(`[A-Za-z]+`)
)

var russianSequences = []struct {
	latin    string
	cyrillic string
}{
	{"shch", "щ"}, {"sch", "щ"}, {"yiy", "ый"}, {"yj", "ый"},
	{"yo", "ё"}, {"jo", "ё"}, {"zh", "ж"}, {"kh", "х"},
	{"ts", "ц"}, {"ch", "ч"}, {"sh", "ш"}, {"yu", "ю"},
	{"ju", "ю"}, {"ya", "я"}, {"ja", "я"},
}

var russianLetters = map[byte]string{
	'a': "а", 'b': "б", 'c': "ц", 'd': "д", 'e': "е", 'f': "ф",
	'g': "г", 'h': "х", 'i': "и", 'j': "й", 'k': "к", 'l': "л",
	'm': "м", 'n': "н", 'o': "о", 'p': "п", 'q': "к", 'r': "р",
	's': "с", 't': "т", 'u': "у", 'v': "в", 'w': "в", 'x': "кс",
	'y': "ы", 'z': "з",
}

// russianTitleVariant reconstructs Cyrillic from common torrent-name
// transliteration. It deliberately requires Russian-looking signals so normal
// English titles are not retried as meaningless Cyrillic queries.
func russianTitleVariant(title string) (string, bool) {
	if strings.IndexFunc(title, func(r rune) bool { return unicode.In(r, unicode.Cyrillic) }) >= 0 || russianTitleScore(title) < 2 {
		return "", false
	}
	return latinWordPattern.ReplaceAllStringFunc(title, transliterateWord), true
}

func russianTitleScore(title string) int {
	lower := strings.ToLower(title)
	score := 0
	for _, signal := range []string{"shch", "sch", "zh", "kh", "yu", "ya", "yo", "ja", "ju", "yj", "yiy", "sht"} {
		if strings.Contains(lower, signal) {
			score += 2
		}
	}
	if strings.Contains(lower, "j") {
		score += 2
	} else if strings.Contains(lower, "y") {
		score++
	}
	for _, word := range strings.FieldsFunc(lower, func(r rune) bool { return !unicode.IsLetter(r) }) {
		switch word {
		case "i", "v", "vo", "s", "so", "k", "ko", "na", "za", "iz", "ot", "do", "ne", "o", "ob":
			score += 2
		}
		if strings.HasSuffix(word, "oe") || strings.HasSuffix(word, "iya") || strings.HasSuffix(word, "ie") || strings.HasSuffix(word, "ej") {
			score++
		}
	}
	return score
}

func transliterateWord(word string) string {
	lower := strings.ToLower(word)
	var result strings.Builder
	for index := 0; index < len(lower); {
		mapped := ""
		consumed := 0
		for _, sequence := range russianSequences {
			if strings.HasPrefix(lower[index:], sequence.latin) {
				mapped, consumed = sequence.cyrillic, len(sequence.latin)
				break
			}
		}
		if consumed == 0 {
			letter := lower[index]
			mapped, consumed = russianLetters[letter], 1
			if letter == 'j' && index == len(lower)-1 && index > 0 && !strings.ContainsRune("aeiouy", rune(lower[index-1])) {
				mapped = "ь"
			}
		}
		if index == 0 && word[0] >= 'A' && word[0] <= 'Z' {
			r, size := utf8.DecodeRuneInString(mapped)
			mapped = string(unicode.ToUpper(r)) + mapped[size:]
		}
		result.WriteString(mapped)
		index += consumed
	}
	return result.String()
}

func lookupTitleVariants(title string) []string {
	variants := make([]string, 0, 4)
	add := func(value string) {
		value = strings.TrimSpace(value)
		for _, existing := range variants {
			if strings.EqualFold(existing, value) {
				return
			}
		}
		if value != "" {
			variants = append(variants, value)
		}
	}
	add(title)
	cleaned := strings.Join(strings.Fields(filmCountPattern.ReplaceAllString(title, " ")), " ")
	add(cleaned)
	if cyrillic, ok := russianTitleVariant(cleaned); ok {
		add(cyrillic)
	}
	if cyrillic, ok := russianTitleVariant(title); ok {
		add(cyrillic)
	}
	return variants
}
