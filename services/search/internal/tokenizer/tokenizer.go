package tokenizer

import (
	"regexp"
	"strings"
)

type Tokenizer struct {
	stopWords map[string]bool
}

type Token struct {
	Word     string
	Position int
}

func NewTokenizer() *Tokenizer {
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true,
		"at": true, "be": true, "by": true, "for": true, "from": true,
		"has": true, "he": true, "in": true, "is": true, "it": true,
		"its": true, "of": true, "on": true, "that": true, "the": true,
		"to": true, "was": true, "will": true, "with": true,
	}

	return &Tokenizer{stopWords: stopWords}
}

func (t *Tokenizer) Tokenize(text string) []Token {
	text = strings.ToLower(text)

	reg := regexp.MustCompile(`[^a-z0-9\s\-']+`)
	text = reg.ReplaceAllString(text, " ")

	words := strings.Fields(text)

	tokens := make([]Token, 0)
	position := 0

	for _, word := range words {
		if len(word) < 3 || t.stopWords[word] {
			continue
		}

		word = strings.Trim(word, "-'")
		if word == "" {
			continue
		}

		stemmed := t.stemConservative(word)

		tokens = append(tokens, Token{
			Word:     stemmed,
			Position: position,
		})
		position++
	}

	return tokens
}

func (t *Tokenizer) stemConservative(word string) string {
	wordLen := len(word)

	if wordLen > 4 && strings.HasSuffix(word, "ies") {
		return word[:wordLen-3] + "y"
	}

	if wordLen > 3 && strings.HasSuffix(word, "es") &&
		!strings.HasSuffix(word, "ies") && !strings.HasSuffix(word, "oes") {
		lastChars := word[wordLen-3:]
		if strings.Contains("sxz", lastChars[:1]) ||
			strings.HasSuffix(lastChars, "ch") ||
			strings.HasSuffix(lastChars, "sh") {
			return word[:wordLen-2]
		}
	}

	if wordLen > 3 && strings.HasSuffix(word, "s") &&
		!strings.HasSuffix(word, "ss") &&
		!strings.HasSuffix(word, "us") &&
		!strings.HasSuffix(word, "is") {
		return word[:wordLen-1]
	}

	if wordLen > 5 && strings.HasSuffix(word, "ing") {
		stem := word[:wordLen-3]
		if strings.ContainsAny(stem, "aeiou") {
			return stem
		}
	}

	if wordLen > 4 && strings.HasSuffix(word, "ed") {
		stem := word[:wordLen-2]
		if strings.ContainsAny(stem, "aeiou") {
			return stem
		}
	}

	return word
}
