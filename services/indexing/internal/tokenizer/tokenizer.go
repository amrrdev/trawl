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

	reg := regexp.MustCompile(`[^a-z0-9\s]+`)
	text = reg.ReplaceAllString(text, " ")

	words := strings.Fields(text)

	tokens := make([]Token, 0)
	position := 0

	for _, word := range words {
		if word == "" || len(word) < 2 || t.stopWords[word] {
			continue
		}

		stemmed := t.stem(word)

		tokens = append(tokens, Token{
			Word:     stemmed,
			Position: position,
		})
		position++
	}

	return tokens
}

func (t *Tokenizer) stem(word string) string {
	// Remove plurals
	if strings.HasSuffix(word, "ies") && len(word) > 4 {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "es") && len(word) > 3 {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") && len(word) > 2 {
		return word[:len(word)-1]
	}

	// Remove -ing, -ed
	if strings.HasSuffix(word, "ing") && len(word) > 5 {
		return word[:len(word)-3]
	}
	if strings.HasSuffix(word, "ed") && len(word) > 4 {
		return word[:len(word)-2]
	}

	return word
}
