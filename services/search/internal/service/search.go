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
	searcher  *Searcher
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
	// create a Scylla client adapter and BM25 searcher (default shard count = 4)
	client := NewScyllaClient(scylla)
	searcher := NewSearcher(client, 4)
	return &Search{
		scylladb:  scylla,
		tokenizer: tokenizer.NewTokenizer(),
		minio:     minio,
		searcher:  searcher,
	}
}

func (s *Search) Search(ctx context.Context, query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []SearchResult{}, nil
	}

	log.Printf("🔍 Search query (BM25): %q", query)

	// Delegate candidate retrieval & scoring to the BM25 Searcher implemented in query.go
	candidates, err := s.searcher.Search(ctx, query, 50)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		log.Printf("⚠️  No candidates returned from searcher for query: %q", query)
		return []SearchResult{}, nil
	}

	var results []SearchResult
	for _, c := range candidates {
		// convert doc id string to UUID for metadata lookup
		id, err := gocql.ParseUUID(c.DocID)
		if err != nil {
			log.Printf("⚠️  invalid doc id from index: %s", c.DocID)
			continue
		}
		doc, err := s.getDocument(ctx, id)
		if err != nil {
			log.Printf("⚠️  Failed to get document %s: %v", id, err)
			continue
		}

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
			DocID:       c.DocID,
			Title:       doc.Title,
			Author:      doc.Author,
			Score:       c.Score,
			DownloadURL: downloadURL,
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > 50 {
		results = results[:50]
	}
	log.Printf("🔍 Generated %d search results (BM25)", len(results))
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
