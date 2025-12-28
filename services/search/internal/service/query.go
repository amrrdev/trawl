package service

import (
	"container/heap"
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/amrrdev/trawl/services/search/internal/tokenizer"
)

type ScyllaClient interface {
	GetPostings(ctx context.Context, shard int, terms []string, topN int) (PostingsResponse, error)
}

type Posting struct {
	DocID     string
	TF        int
	Positions []int
}

type PostingsResponse struct {
	ShardID  int
	Results  []DocScore
	DocCount int
}

type DocScore struct {
	DocID   string
	Score   float64
	TF      int
	DocLen  int
	DocFreq int
}

type Searcher struct {
	Client     ScyllaClient
	ShardCount int
	K1         float64
	B          float64
}

func NewSearcher(client ScyllaClient, shards int) *Searcher {
	return &Searcher{
		Client:     client,
		ShardCount: shards,
		K1:         1.2,
		B:          0.75,
	}
}

func (s *Searcher) routeTerms(terms []string) map[int][]string {
	m := make(map[int][]string)
	for _, t := range terms {
		h := int(hashString(t)) % s.ShardCount
		m[h] = append(m[h], t)
	}
	return m
}

func (s *Searcher) Search(ctx context.Context, query string, topK int) ([]DocScore, error) {
	// use the project's tokenizer to normalize, lowercase and stem terms
	tk := tokenizer.NewTokenizer()
	toks := tk.Tokenize(query)
	var terms []string
	for _, t := range toks {
		terms = append(terms, t.Word)
	}
	termToShards := s.routeTerms(terms)
	type shardResult struct {
		resp PostingsResponse
		err  error
	}
	resultsCh := make(chan shardResult, len(termToShards))
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	for shard, termsForShard := range termToShards {
		wg.Add(1)
		go func(sh int, ts []string) {
			defer wg.Done()
			resp, err := s.Client.GetPostings(ctx, sh, ts, topK*2)
			if err != nil {
				resultsCh <- shardResult{err: err}
				return
			}
			resultsCh <- shardResult{resp: resp}
		}(shard, termsForShard)
	}
	go func() {
		wg.Wait()
		close(resultsCh)
	}()
	var shardResponses []PostingsResponse
	for r := range resultsCh {
		if r.err != nil {
			return nil, fmt.Errorf("shard fetch error: %w", r.err)
		}
		shardResponses = append(shardResponses, r.resp)
	}
	merged := mergeShardCandidates(shardResponses, topK)
	return merged, nil
}

func mergeShardCandidates(shardResponses []PostingsResponse, topK int) []DocScore {
	var all []DocScore
	totalDocs := 0
	totalDocLen := 0
	docCount := 0
	for _, sr := range shardResponses {
		totalDocs += sr.DocCount
		for _, d := range sr.Results {
			totalDocLen += d.DocLen
			docCount++
		}
	}
	avgDocLen := 1.0
	if docCount > 0 {
		avgDocLen = float64(totalDocLen) / float64(docCount)
	}
	for _, sr := range shardResponses {
		for _, d := range sr.Results {
			score := bm25Score(d.TF, d.DocLen, avgDocLen, d.DocFreq, totalDocs, 1.2, 0.75)
			all = append(all, DocScore{DocID: d.DocID, Score: score, TF: d.TF, DocLen: d.DocLen, DocFreq: d.DocFreq})
		}
	}
	h := &minHeap{}
	heap.Init(h)
	for _, d := range all {
		if h.Len() < topK {
			heap.Push(h, d)
			continue
		}
		if d.Score > (*h)[0].Score {
			heap.Pop(h)
			heap.Push(h, d)
		}
	}
	n := h.Len()
	out := make([]DocScore, n)
	for i := n - 1; i >= 0; i-- {
		out[i] = heap.Pop(h).(DocScore)
	}
	return out
}

func hashString(s string) uint64 {
	var h uint64 = 1469598103934665603
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

func bm25Score(tf int, docLen int, avgDocLen float64, docFreq int, totalDocs int, k1, b float64) float64 {
	if tf == 0 || docFreq == 0 {
		return 0
	}
	idf := math.Log((float64(totalDocs)-float64(docFreq)+0.5)/(float64(docFreq)+0.5) + 1)
	tfNorm := float64(tf) * (k1 + 1) / (float64(tf) + k1*(1-b+b*(float64(docLen)/avgDocLen)))
	return idf * tfNorm
}

type minHeap []DocScore

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].Score < h[j].Score }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x interface{}) {
	*h = append(*h, x.(DocScore))
}
func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
