package service

import (
	"context"
	"sort"

	"github.com/amrrdev/trawl/services/search/internal/scylladb"
	"github.com/gocql/gocql"
)

// ScyllaClientImpl implements the ScyllaClient interface using the project's ScyllaDB wrapper.
type ScyllaClientImpl struct {
	db *scylladb.ScyllaDB
}

func NewScyllaClient(db *scylladb.ScyllaDB) *ScyllaClientImpl {
	return &ScyllaClientImpl{db: db}
}

func (c *ScyllaClientImpl) GetPostings(ctx context.Context, shard int, terms []string, topN int) (PostingsResponse, error) {
	var results []DocScore
	totalDocs := 0

	for _, term := range terms {
		// Try to read doc_count from word_stats (counter table). If missing, fallback to counting inverted_index rows.
		var docCount int
		if err := c.db.Session.Query(`SELECT doc_count FROM word_stats WHERE word = ?`, term).WithContext(ctx).Scan(&docCount); err != nil {
			// fallback: count rows for the term
			iter := c.db.Session.Query(`SELECT doc_id FROM inverted_index WHERE word = ?`, term).WithContext(ctx).Iter()
			var id gocql.UUID
			seen := make(map[string]struct{})
			for iter.Scan(&id) {
				seen[id.String()] = struct{}{}
			}
			_ = iter.Close()
			docCount = len(seen)
		}

		totalDocs += docCount

		// Fetch postings for the term
		iter := c.db.Session.Query(`SELECT doc_id, term_frequency, positions FROM inverted_index WHERE word = ?`, term).WithContext(ctx).Iter()
		var docID gocql.UUID
		var tf int
		var positions []int
		for iter.Scan(&docID, &tf, &positions) {
			ds := DocScore{
				DocID:   docID.String(),
				TF:      tf,
				DocLen:  len(positions),
				DocFreq: docCount,
			}
			results = append(results, ds)
		}
		if err := iter.Close(); err != nil {
			return PostingsResponse{}, err
		}
	}

	// Keep topN results per shard by score proxy (TF) to limit data transferred.
	sort.Slice(results, func(i, j int) bool { return results[i].TF > results[j].TF })
	if len(results) > topN {
		results = results[:topN]
	}

	return PostingsResponse{ShardID: shard, Results: results, DocCount: totalDocs}, nil
}
