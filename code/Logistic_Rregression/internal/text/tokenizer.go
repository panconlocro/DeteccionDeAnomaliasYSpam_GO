package text

import (
	"strings"
	"unicode"
)

var accentReplacer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
	"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u",
	"à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u",
	"À", "a", "È", "e", "Ì", "i", "Ò", "o", "Ù", "u",
	"ä", "a", "ë", "e", "ï", "i", "ö", "o", "ü", "u",
	"Ä", "a", "Ë", "e", "Ï", "i", "Ö", "o", "Ü", "u",
	"ñ", "n", "Ñ", "n",
)

func Tokenize(value string, useBigrams bool) []string {
	value = Normalize(value)
	unigrams := strings.Fields(value)
	if !useBigrams || len(unigrams) < 2 {
		return unigrams
	}

	tokens := make([]string, 0, len(unigrams)*2-1)
	tokens = append(tokens, unigrams...)
	for i := 0; i < len(unigrams)-1; i++ {
		tokens = append(tokens, unigrams[i]+"_"+unigrams[i+1])
	}
	return tokens
}

func Normalize(value string) string {
	value = accentReplacer.Replace(value)
	value = strings.ToLower(value)

	var b strings.Builder
	b.Grow(len(value))

	lastWasSpace := true
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastWasSpace = false
		case unicode.IsSpace(r):
			if !lastWasSpace {
				b.WriteByte(' ')
				lastWasSpace = true
			}
		default:
			if !lastWasSpace {
				b.WriteByte(' ')
				lastWasSpace = true
			}
		}
	}

	return strings.TrimSpace(b.String())
}
