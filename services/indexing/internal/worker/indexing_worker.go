package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/amrrdev/trawl/services/indexing/internal/parser"
	"github.com/amrrdev/trawl/services/indexing/internal/queue"
	"github.com/amrrdev/trawl/services/indexing/internal/scylladb"
	"github.com/amrrdev/trawl/services/indexing/internal/tokenizer"
	"github.com/amrrdev/trawl/services/indexing/internal/types"
	"github.com/amrrdev/trawl/services/shared/storage"
	"github.com/gocql/gocql"
	"github.com/minio/minio-go/v7"
	amqp "github.com/rabbitmq/amqp091-go"
)

type IndexingWorker struct {
	consumer       *queue.Consumer
	minioStorage   *storage.Storage
	tokenizer      *tokenizer.Tokenizer
	scylladb       *scylladb.ScyllaDB
	parserRegistry *parser.Registry
	concurrency    int
	batchSize      int
	maxRetries     int
}

func NewIndexingWorker(
	consumer *queue.Consumer,
	minioStorage *storage.Storage,
	scylla *scylladb.ScyllaDB,
) *IndexingWorker {
	return &IndexingWorker{
		consumer:       consumer,
		scylladb:       scylla,
		minioStorage:   minioStorage,
		tokenizer:      tokenizer.NewTokenizer(),
		parserRegistry: parser.NewRegistry(),
		concurrency:    5,
		batchSize:      50,
		maxRetries:     3,
	}
}

func (w *IndexingWorker) Start(ctx context.Context) error {
	log.Printf("Starting indexing worker with %d concurrent workers", w.concurrency)

	messages, err := w.consumer.Consume()
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < w.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.worker(ctx, workerID, messages)
		}(i)
	}

	<-ctx.Done()
	log.Println("Shutting down workers...")

	wg.Wait()

	return ctx.Err()
}

func (w *IndexingWorker) worker(ctx context.Context, workerID int, messages <-chan amqp.Delivery) {
	log.Printf("Worker %d started", workerID)

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				log.Printf("Worker %d stopped (channel closed)", workerID)
				return
			}

			var job types.IndexingJob
			if err := json.Unmarshal(msg.Body, &job); err != nil {
				log.Printf("Worker %d: Failed to parse job: %v", workerID, err)
				msg.Nack(false, false)
				continue
			}

			if err := w.processJob(ctx, workerID, &job); err != nil {
				log.Printf("Worker %d: Failed to process job %s: %v", workerID, job.JobID, err)

				retryCount := w.getRetryCount(msg)
				if retryCount < w.maxRetries {
					retryCount++
					log.Printf("Worker %d: Retrying job %s (attempt %d/%d)",
						workerID, job.JobID, retryCount, w.maxRetries)
					if msg.Headers == nil {
						msg.Headers = make(map[string]interface{})
					}
					msg.Headers["x-retry-count"] = int32(retryCount)
					if pubErr := w.consumer.Publish(msg.Body, msg.Headers); pubErr != nil {
						log.Printf("Worker %d: Failed to republish job %s: %v", workerID, job.JobID, pubErr)
						msg.Nack(false, false)
					} else {
						msg.Ack(false)
					}
				} else {
					log.Printf("Worker %d: Job %s failed after %d retries, sending to DLQ",
						workerID, job.JobID, w.maxRetries)
					msg.Nack(false, false)
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				log.Printf("Worker %d: Failed to ack message: %v", workerID, err)
			}

		case <-ctx.Done():
			log.Printf("Worker %d stopped (context cancelled)", workerID)
			return
		}
	}
}

func (w *IndexingWorker) getRetryCount(msg amqp.Delivery) int {
	if msg.Headers == nil {
		return 0
	}

	if count, ok := msg.Headers["x-retry-count"].(int32); ok {
		return int(count)
	}

	return 0
}

func (w *IndexingWorker) processJob(ctx context.Context, workerID int, job *types.IndexingJob) error {
	startTime := time.Now()
	log.Printf("Worker %d: Processing job %s (doc: %s)", workerID, job.JobID, job.Payload.DocID)

	parsedDoc, err := w.downloadAndParse(ctx, job.Payload.FilePath)
	if err != nil {
		return fmt.Errorf("failed to parse document: %w", err)
	}

	tokens := w.tokenizer.Tokenize(parsedDoc.Content)
	log.Printf("Worker %d: Extracted %d tokens from document %s", workerID, len(tokens), job.Payload.DocID)

	if len(tokens) == 0 {
		return fmt.Errorf("no tokens extracted from document")
	}

	if err := w.buildInvertedIndex(ctx, job.Payload.DocID, tokens); err != nil {
		return fmt.Errorf("failed to build inverted index: %w", err)
	}

	if err := w.storeDocumentMetadata(ctx, job, parsedDoc, len(tokens)); err != nil {
		return fmt.Errorf("failed to store document metadata: %w", err)
	}

	go func() {
		statsCtx := context.Background()
		if err := w.updateWordStats(statsCtx, tokens); err != nil {
			log.Printf("Worker %d: Failed to update word stats (non-critical): %v", workerID, err)
		} else {
			log.Printf("Worker %d: Updated word statistics", workerID)
		}
	}()

	duration := time.Since(startTime)
	log.Printf("Worker %d: Successfully indexed document %s in %v", workerID, job.Payload.DocID, duration)
	return nil
}

func (w *IndexingWorker) downloadAndParse(ctx context.Context, filePath string) (*parser.ParsedDocument, error) {
	reader, err := w.minioStorage.Client.GetObject(ctx, w.minioStorage.Bucket, filePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer reader.Close()

	parsedDoc, err := w.parserRegistry.ParseFile(ctx, filePath, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	return parsedDoc, nil
}

func (w *IndexingWorker) buildInvertedIndex(ctx context.Context, docID string, tokens []tokenizer.Token) error {
	wordMap := make(map[string]*WordData)

	for _, token := range tokens {
		if data, exists := wordMap[token.Word]; exists {
			data.Positions = append(data.Positions, token.Position)
			data.Frequency++
		} else {
			wordMap[token.Word] = &WordData{
				Word:      token.Word,
				Positions: []int{token.Position},
				Frequency: 1,
			}
		}
	}

	words := make([]*WordData, 0, len(wordMap))
	for _, data := range wordMap {
		words = append(words, data)
	}

	return w.insertWordsBatched(ctx, docID, words)
}

func (w *IndexingWorker) insertWordsBatched(ctx context.Context, docID string, words []*WordData) error {
	numBatches := (len(words) + w.batchSize - 1) / w.batchSize
	errChan := make(chan error, numBatches)
	var wg sync.WaitGroup

	for i := 0; i < len(words); i += w.batchSize {
		end := i + w.batchSize
		if end > len(words) {
			end = len(words)
		}

		batch := words[i:end]
		wg.Add(1)

		go func(batchWords []*WordData) {
			defer wg.Done()
			if err := w.insertBatch(ctx, docID, batchWords); err != nil {
				errChan <- err
			}
		}(batch)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *IndexingWorker) insertBatch(ctx context.Context, docID string, words []*WordData) error {
	batch := w.scylladb.Session.NewBatch(gocql.LoggedBatch)

	docUUID, err := gocql.ParseUUID(docID)
	if err != nil {
		return fmt.Errorf("invalid doc_id UUID: %w", err)
	}

	for _, word := range words {
		query := `
            INSERT INTO inverted_index (word, doc_id, term_frequency, positions)
            VALUES (?, ?, ?, ?)
        `
		batch.Query(query, word.Word, docUUID, word.Frequency, word.Positions)
	}

	if err := w.scylladb.Session.ExecuteBatch(batch.WithContext(ctx)); err != nil {
		return fmt.Errorf("batch insert failed: %w", err)
	}

	return nil
}

func (w *IndexingWorker) storeDocumentMetadata(
	ctx context.Context,
	job *types.IndexingJob,
	parsedDoc *parser.ParsedDocument,
	wordCount int,
) error {
	docUUID, err := gocql.ParseUUID(job.Payload.DocID)
	if err != nil {
		return fmt.Errorf("invalid doc_id UUID: %w", err)
	}

	title := parsedDoc.Metadata["title"]
	if title == "" {
		title = job.Payload.FileName
	}

	author := parsedDoc.Metadata["author"]
	if author == "" {
		author = "unknown"
	}

	query := `
        INSERT INTO documents (doc_id, title, author, file_path, created_at)
        VALUES (?, ?, ?, ?, ?)
    `

	return w.scylladb.Session.Query(query,
		docUUID,
		title,
		author,
		job.Payload.FilePath,
		time.Now(),
	).WithContext(ctx).Exec()
}

func (w *IndexingWorker) updateWordStats(ctx context.Context, tokens []tokenizer.Token) error {
	uniqueWords := make(map[string]int)
	for _, token := range tokens {
		uniqueWords[token.Word]++
	}

	const batchSize = 100
	var wg sync.WaitGroup
	errChan := make(chan error, (len(uniqueWords)+batchSize-1)/batchSize)

	wordList := make([]string, 0, len(uniqueWords))
	freqList := make([]int, 0, len(uniqueWords))
	for word, freq := range uniqueWords {
		wordList = append(wordList, word)
		freqList = append(freqList, freq)
	}

	for i := 0; i < len(wordList); i += batchSize {
		end := i + batchSize
		if end > len(wordList) {
			end = len(wordList)
		}
		batchWords := wordList[i:end]
		batchFreqs := freqList[i:end]

		wg.Add(1)
		go func(words []string, freqs []int) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := w.updateWordStatsBatch(ctx, words, freqs); err != nil {
				errChan <- err
			}
		}(batchWords, batchFreqs)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *IndexingWorker) updateWordStatsBatch(ctx context.Context, words []string, freqs []int) error {
	for i, word := range words {
		query := `
            UPDATE word_stats
            SET doc_count = doc_count + 1,
                total_occurrences = total_occurrences + ?
            WHERE word = ?
        `
		if err := w.scylladb.Session.Query(query, freqs[i], word).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf("failed to update stats for word %q: %w", word, err)
		}
	}
	return nil
}

type WordData struct {
	Word      string
	Positions []int
	Frequency int
}
