package service

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/amrrdev/trawl/services/search/internal/scylladb"
	"github.com/amrrdev/trawl/services/search/internal/tokenizer"
	"github.com/amrrdev/trawl/services/shared/storage"
	"github.com/gocql/gocql"
)

type Search struct {
	scylladb  *scylladb.ScyllaDB
	tokenizer *tokenizer.Tokenizer
	minio     *storage.Storage
}

type SearchResult struct {
	DocID       string  `json:"doc_id"`
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	Score       float64 `json:"score"`
	Snippet     string  `json:"snippet,omitempty"`
	DownloadURL string  `json:"download_url"`
}

func NewSearch(scylla *scylladb.ScyllaDB, minio *storage.Storage) *Search {
	return &Search{
		scylladb:  scylla,
		tokenizer: tokenizer.NewTokenizer(),
		minio:     minio,
	}
}

func (s *Search) Search(ctx context.Context, query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []SearchResult{}, nil
	}

	log.Printf("🔍 Search query: %q", query)

	tokens := s.tokenizer.Tokenize(query)
	log.Printf("🔍 Tokenized to %d tokens: %v", len(tokens), tokens)

	if len(tokens) == 0 {
		log.Printf("⚠️  No valid tokens found in query")
		return []SearchResult{}, nil
	}

	uniqueTokens := make(map[string]bool)
	var filteredTokens []tokenizer.Token
	for _, token := range tokens {
		if !uniqueTokens[token.Word] {
			uniqueTokens[token.Word] = true
			filteredTokens = append(filteredTokens, token)
		}
	}

	validTokens := 0
	for _, token := range filteredTokens {
		exists, err := s.tokenExistsInIndex(ctx, token.Word)
		if err != nil {
			log.Printf("⚠️  Error checking if token %q exists: %v", token.Word, err)
			continue
		}
		if exists {
			validTokens++
		} else {
			log.Printf("⚠️  Token %q not found in index", token.Word)
		}
	}

	if validTokens == 0 {
		log.Printf("⚠️  No valid tokens found in index for query: %q", query)
		return []SearchResult{}, nil
	}

	log.Printf("🔍 Found %d/%d tokens in index", validTokens, len(filteredTokens))

	docScores := make(map[gocql.UUID]float64)
	docMatches := make(map[gocql.UUID][]string)
	totalDocsFound := 0

	for _, token := range filteredTokens {
		docs, err := s.queryInvertedIndex(ctx, token.Word)
		if err != nil {
			log.Printf("⚠️  Failed to query inverted index for token %q: %v", token.Word, err)
			continue
		}

		log.Printf("🔍 Token %q found in %d documents", token.Word, len(docs))
		totalDocsFound += len(docs)

		for _, doc := range docs {
			tfScore := float64(doc.Frequency)
			docScores[doc.DocID] += tfScore
			docMatches[doc.DocID] = append(docMatches[doc.DocID], token.Word)
		}
	}

	log.Printf("🔍 Total documents found across all tokens: %d", totalDocsFound)
	log.Printf("🔍 Unique documents with matches: %d", len(docScores))

	if len(docScores) == 0 {
		log.Printf("⚠️  No documents found matching the query")
		return []SearchResult{}, nil
	}

	var results []SearchResult
	for docID, score := range docScores {
		doc, err := s.getDocument(ctx, docID)
		if err != nil {
			log.Printf("⚠️  Failed to get document %s: %v", docID, err)
			continue
		}

		matchCount := len(docMatches[docID])
		finalScore := score * float64(matchCount)

		downloadURL := ""
		if doc.FilePath != "" {
			url, err := s.minio.GetDownloadUrl(ctx, doc.UserID, doc.FileName, 24*time.Hour)
			if err != nil {
				log.Printf("⚠️  Failed to generate download URL for %s: %v", doc.FileName, err)
			} else {
				downloadURL = url
			}
		}

		results = append(results, SearchResult{
			DocID:       docID.String(),
			Title:       doc.Title,
			Author:      doc.Author,
			Score:       finalScore,
			DownloadURL: downloadURL,
		})
	}

	log.Printf("🔍 Generated %d search results", len(results))

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > 50 {
		results = results[:50]
	}

	return results, nil
}

func (s *Search) tokenExistsInIndex(ctx context.Context, word string) (bool, error) {
	query := `SELECT word FROM inverted_index WHERE word = ? LIMIT 1`
	iter := s.scylladb.Session.Query(query, word).WithContext(ctx).Iter()

	var foundWord string
	hasNext := iter.Scan(&foundWord)

	err := iter.Close()
	if err != nil {
		return false, err
	}

	return hasNext, nil
}

type invertedIndexResult struct {
	DocID     gocql.UUID
	Frequency int
	Positions []int
}

func (s *Search) queryInvertedIndex(ctx context.Context, word string) ([]invertedIndexResult, error) {
	query := `SELECT doc_id, term_frequency, positions FROM inverted_index WHERE word = ?`
	iter := s.scylladb.Session.Query(query, word).WithContext(ctx).Iter()

	var results []invertedIndexResult
	var docID gocql.UUID
	var frequency int
	var positions []int

	for iter.Scan(&docID, &frequency, &positions) {
		results = append(results, invertedIndexResult{
			DocID:     docID,
			Frequency: frequency,
			Positions: positions,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return results, nil
}

type documentResult struct {
	Title    string
	Author   string
	FilePath string
	UserID   string
	FileName string
}

func (s *Search) getDocument(ctx context.Context, docID gocql.UUID) (*documentResult, error) {
	query := `SELECT title, author, file_path FROM documents WHERE doc_id = ?`
	var title, author, filePath string

	err := s.scylladb.Session.Query(query, docID).WithContext(ctx).Scan(&title, &author, &filePath)
	if err != nil {
		return nil, err
	}

	// Parse file_path to extract userID and fileName
	// file_path format: "userID/filename"
	userID := ""
	fileName := ""
	if filePath != "" {
		parts := strings.Split(filePath, "/")
		if len(parts) >= 2 {
			userID = parts[0]
			fileName = strings.Join(parts[1:], "/") // Handle filenames with slashes
		}
	}

	return &documentResult{
		Title:    title,
		Author:   author,
		FilePath: filePath,
		UserID:   userID,
		FileName: fileName,
	}, nil
}
