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
		batchSize:      1000,
		maxRetries:     3,
	}
}

func (w *IndexingWorker) Start(ctx context.Context) error {
	log.Printf("🚀 Starting indexing worker with %d concurrent workers", w.concurrency)

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

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("⏹️  Shutting down workers...")

	// Wait for all workers to finish current jobs
	wg.Wait()

	return ctx.Err()
}

// worker processes messages from the RabbitMQ channel
func (w *IndexingWorker) worker(ctx context.Context, workerID int, messages <-chan amqp.Delivery) {
	log.Printf("👷 Worker %d started", workerID)

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				log.Printf("👷 Worker %d stopped (channel closed)", workerID)
				return
			}

			// Parse job from message
			var job types.IndexingJob
			if err := json.Unmarshal(msg.Body, &job); err != nil {
				log.Printf("❌ Worker %d: Failed to parse job: %v", workerID, err)
				msg.Nack(false, false) // Send to DLQ
				continue
			}

			// Process the job
			if err := w.processJob(ctx, workerID, &job); err != nil {
				log.Printf("❌ Worker %d: Failed to process job %s: %v", workerID, job.JobID, err)

				// Check retry count and republish with incremented header
				retryCount := w.getRetryCount(msg)
				if retryCount < w.maxRetries {
					retryCount++
					log.Printf("🔄 Worker %d: Retrying job %s (attempt %d/%d)",
						workerID, job.JobID, retryCount, w.maxRetries)
					// Republish with updated header instead of requeueing
					if msg.Headers == nil {
						msg.Headers = make(map[string]interface{})
					}
					msg.Headers["x-retry-count"] = int32(retryCount)
					if pubErr := w.consumer.Publish(msg.Body, msg.Headers); pubErr != nil {
						log.Printf("❌ Worker %d: Failed to republish job %s: %v", workerID, job.JobID, pubErr)
						msg.Nack(false, false) // Send to DLQ on publish failure
					} else {
						msg.Ack(false) // Acknowledge original message after republishing
					}
				} else {
					log.Printf("💀 Worker %d: Job %s failed after %d retries, sending to DLQ",
						workerID, job.JobID, w.maxRetries)
					msg.Nack(false, false) // Send to DLQ
				}
				continue
			}

			// Success - acknowledge message
			if err := msg.Ack(false); err != nil {
				log.Printf("⚠️  Worker %d: Failed to ack message: %v", workerID, err)
			}

		case <-ctx.Done():
			log.Printf("👷 Worker %d stopped (context cancelled)", workerID)
			return
		}
	}
}

// getRetryCount extracts retry count from message headers
func (w *IndexingWorker) getRetryCount(msg amqp.Delivery) int {
	if msg.Headers == nil {
		return 0
	}

	if count, ok := msg.Headers["x-retry-count"].(int32); ok {
		return int(count)
	}

	return 0
}

// processJob handles a single indexing job
func (w *IndexingWorker) processJob(ctx context.Context, workerID int, job *types.IndexingJob) error {
	startTime := time.Now()
	log.Printf("📄 Worker %d: Processing job %s (doc: %s)", workerID, job.JobID, job.Payload.DocID)

	// 1. Download and parse file
	parsedDoc, err := w.downloadAndParse(ctx, job.Payload.FilePath)
	if err != nil {
		return fmt.Errorf("failed to parse document: %w", err)
	}

	// 2. Tokenize text
	tokens := w.tokenizer.Tokenize(parsedDoc.Content)
	log.Printf("✅ Worker %d: Extracted %d tokens from document %s", workerID, len(tokens), job.Payload.DocID)

	if len(tokens) == 0 {
		return fmt.Errorf("no tokens extracted from document")
	}

	// 3. Build inverted index (parallel batching)
	if err := w.buildInvertedIndex(ctx, job.Payload.DocID, tokens); err != nil {
		return fmt.Errorf("failed to build inverted index: %w", err)
	}

	// 4. Store document metadata
	if err := w.storeDocumentMetadata(ctx, job, parsedDoc, len(tokens)); err != nil {
		return fmt.Errorf("failed to store document metadata: %w", err)
	}

	// 5. Update word statistics (async - don't wait)
	go func() {
		statsCtx := context.Background()
		if err := w.updateWordStats(statsCtx, tokens); err != nil {
			log.Printf("⚠️  Worker %d: Failed to update word stats: %v", workerID, err)
		}
	}()

	duration := time.Since(startTime)
	log.Printf("✅ Worker %d: Successfully indexed document %s in %v", workerID, job.Payload.DocID, duration)
	return nil
}

// downloadAndParse downloads file from MinIO and extracts text
func (w *IndexingWorker) downloadAndParse(ctx context.Context, filePath string) (*parser.ParsedDocument, error) {
	// Download file from MinIO
	reader, err := w.minioStorage.Client.GetObject(ctx, w.minioStorage.Bucket, filePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer reader.Close()

	// Parse file using registry
	parsedDoc, err := w.parserRegistry.ParseFile(ctx, filePath, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	return parsedDoc, nil
}

// buildInvertedIndex inserts tokens into ScyllaDB with batching
func (w *IndexingWorker) buildInvertedIndex(ctx context.Context, docID string, tokens []tokenizer.Token) error {
	// Group tokens by word
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

	// Convert to slice for parallel processing
	words := make([]*WordData, 0, len(wordMap))
	for _, data := range wordMap {
		words = append(words, data)
	}

	// Insert in parallel batches
	return w.insertWordsBatched(ctx, docID, words)
}

// insertWordsBatched inserts words in parallel batches
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

	// Wait for all batches
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// insertBatch inserts a single batch into ScyllaDB
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

// storeDocumentMetadata stores document info in ScyllaDB
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
		title = job.Payload.FileName // fallback to filename
	}

	author := parsedDoc.Metadata["author"]
	if author == "" {
		author = "unknown" // default
	}

	query := `
        INSERT INTO documents (doc_id, title, author, created_at)
        VALUES (?, ?, ?, ?)
    `

	return w.scylladb.Session.Query(query,
		docUUID,
		title,
		author,
		time.Now(),
	).WithContext(ctx).Exec()
}

// updateWordStats updates global word statistics
func (w *IndexingWorker) updateWordStats(ctx context.Context, tokens []tokenizer.Token) error {
	// Count unique words
	uniqueWords := make(map[string]int)
	for _, token := range tokens {
		uniqueWords[token.Word]++
	}

	// Batch updates into groups of 100 for efficiency
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
				return // Respect context cancellation
			default:
			}
			if err := w.updateWordStatsBatch(ctx, words, freqs); err != nil {
				errChan <- err
			}
		}(batchWords, batchFreqs)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *IndexingWorker) updateWordStatsBatch(ctx context.Context, words []string, freqs []int) error {
	batch := w.scylladb.Session.NewBatch(gocql.LoggedBatch)
	for i, word := range words {
		query := `
            UPDATE word_stats
            SET doc_count = doc_count + 1,
                total_occurrences = total_occurrences + ?
            WHERE word = ?
        `
		batch.Query(query, freqs[i], word)
	}
	return w.scylladb.Session.ExecuteBatch(batch.WithContext(ctx))
}

type WordData struct {
	Word      string
	Positions []int
	Frequency int
}
