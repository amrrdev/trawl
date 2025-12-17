package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ledongthuc/pdf"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PDFMessage struct {
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
	ID        string    `json:"id"`
}

type AutoScalingConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	msgs    <-chan amqp.Delivery

	// Worker pool management
	minWorkers    int
	maxWorkers    int
	activeWorkers int32 // atomic counter
	idleWorkers   int32 // atomic counter

	// Auto-scaling config
	scaleUpThreshold int           // Messages in queue to trigger scale-up
	scaleDownIdle    time.Duration // Time idle before scaling down
	checkInterval    time.Duration

	// Channels for worker coordination
	taskChan     chan amqp.Delivery
	workerWg     sync.WaitGroup
	scalingMutex sync.Mutex

	// Stats
	totalProcessed int64
	totalFailed    int64
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewAutoScalingConsumer(amqpURL string, minWorkers, maxWorkers int) (*AutoScalingConsumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &AutoScalingConsumer{
		conn:             conn,
		channel:          ch,
		minWorkers:       minWorkers,
		maxWorkers:       maxWorkers,
		activeWorkers:    0,
		scaleUpThreshold: 10, // Scale up if 10+ messages waiting
		scaleDownIdle:    30 * time.Second,
		checkInterval:    5 * time.Second,
		taskChan:         make(chan amqp.Delivery, 100),
		ctx:              ctx,
		cancel:           cancel,
	}, nil
}

func (c *AutoScalingConsumer) Start() error {
	// Declare queue
	queue, err := c.channel.QueueDeclare(
		"pdf_processing",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Set high prefetch for dynamic scaling
	err = c.channel.Qos(c.maxWorkers, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Register consumer
	msgs, err := c.channel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	c.msgs = msgs

	log.Printf("🚀 Auto-scaling consumer started")
	log.Printf("📊 Min workers: %d, Max workers: %d", c.minWorkers, c.maxWorkers)

	// Start with minimum workers
	for i := 0; i < c.minWorkers; i++ {
		c.spawnWorker()
	}

	// Start message dispatcher
	go c.messageDispatcher()

	// Start auto-scaler (monitors and adjusts workers)
	go c.autoScaler()

	// Start stats reporter
	go c.statsReporter()

	return nil
}

// messageDispatcher receives from RabbitMQ and sends to worker pool
func (c *AutoScalingConsumer) messageDispatcher() {
	for {
		select {
		case <-c.ctx.Done():
			close(c.taskChan)
			return
		case d, ok := <-c.msgs:
			if !ok {
				close(c.taskChan)
				return
			}
			// Send to worker pool (blocks if all workers are busy)
			select {
			case c.taskChan <- d:
				// Message dispatched
			case <-c.ctx.Done():
				return
			}
		}
	}
}

// autoScaler monitors load and adjusts worker count
func (c *AutoScalingConsumer) autoScaler() {
	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.evaluateScaling()
		}
	}
}

func (c *AutoScalingConsumer) evaluateScaling() {
	c.scalingMutex.Lock()
	defer c.scalingMutex.Unlock()

	active := atomic.LoadInt32(&c.activeWorkers)
	idle := atomic.LoadInt32(&c.idleWorkers)
	queueLen := len(c.taskChan)

	// SCALE UP: If queue is building up and we have capacity
	if queueLen > c.scaleUpThreshold && int(active) < c.maxWorkers {
		needed := queueLen / 5 // Add 1 worker per 5 queued messages
		if needed < 1 {
			needed = 1
		}

		toSpawn := min(needed, c.maxWorkers-int(active))

		log.Printf("📈 SCALE UP: Queue has %d messages, spawning %d workers (active: %d -> %d)",
			queueLen, toSpawn, active, active+int32(toSpawn))

		for i := 0; i < toSpawn; i++ {
			c.spawnWorker()
		}
	}

	// SCALE DOWN: If too many workers are idle
	if idle > int32(c.minWorkers) && int(active) > c.minWorkers {
		excessIdle := idle - int32(c.minWorkers)
		toRemove := min(int(excessIdle), int(active)-c.minWorkers)

		log.Printf("📉 SCALE DOWN: %d workers idle, removing %d workers (active: %d -> %d)",
			idle, toRemove, active, active-int32(toRemove))

		// Signal workers to stop (they'll check in their idle loop)
		// This is handled by workers timing out on taskChan receive
	}
}

func (c *AutoScalingConsumer) spawnWorker() {
	workerID := atomic.AddInt32(&c.activeWorkers, 1)
	c.workerWg.Add(1)

	go c.worker(int(workerID))
}

func (c *AutoScalingConsumer) worker(workerID int) {
	defer c.workerWg.Done()
	defer atomic.AddInt32(&c.activeWorkers, -1)

	log.Printf("👷 Worker %d started", workerID)

	idleTimer := time.NewTimer(c.scaleDownIdle)
	defer idleTimer.Stop()

	for {
		// Mark as idle
		atomic.AddInt32(&c.idleWorkers, 1)

		select {
		case <-c.ctx.Done():
			atomic.AddInt32(&c.idleWorkers, -1)
			log.Printf("👷 Worker %d stopped (context cancelled)", workerID)
			return

		case d, ok := <-c.taskChan:
			// Mark as busy
			atomic.AddInt32(&c.idleWorkers, -1)

			if !ok {
				log.Printf("👷 Worker %d stopped (channel closed)", workerID)
				return
			}

			// Reset idle timer (we got work)
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(c.scaleDownIdle)

			c.processMessage(workerID, d)

		case <-idleTimer.C:
			// Been idle too long, check if we should shut down
			active := atomic.LoadInt32(&c.activeWorkers)
			if int(active) > c.minWorkers {
				atomic.AddInt32(&c.idleWorkers, -1)
				log.Printf("👷 Worker %d stopped (idle timeout, active: %d)", workerID, active-1)
				return
			}
			// Reset timer if we need to stay alive
			idleTimer.Reset(c.scaleDownIdle)
		}
	}
}

func (c *AutoScalingConsumer) processMessage(workerID int, d amqp.Delivery) {
	startTime := time.Now()

	var pdfMsg PDFMessage
	err := json.Unmarshal(d.Body, &pdfMsg)
	if err != nil {
		log.Printf("❌ Worker %d: Parse failed: %v", workerID, err)
		d.Nack(false, false)
		atomic.AddInt64(&c.totalFailed, 1)
		return
	}

	text, err := downloadAndExtractPDF(pdfMsg.URL)
	if err != nil {
		log.Printf("❌ Worker %d: Process failed %s: %v", workerID, pdfMsg.ID, err)
		d.Nack(false, true)
		atomic.AddInt64(&c.totalFailed, 1)
		return
	}

	duration := time.Since(startTime)
	log.Printf("✅ Worker %d: Completed %s in %v", workerID, pdfMsg.ID, duration)

	d.Ack(false)
	atomic.AddInt64(&c.totalProcessed, 1)
}

func (c *AutoScalingConsumer) statsReporter() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			active := atomic.LoadInt32(&c.activeWorkers)
			idle := atomic.LoadInt32(&c.idleWorkers)
			processed := atomic.LoadInt64(&c.totalProcessed)
			failed := atomic.LoadInt64(&c.totalFailed)
			queueLen := len(c.taskChan)

			log.Println("📊 ==================== Stats ====================")
			log.Printf("   👷 Active Workers: %d (idle: %d, busy: %d)", active, idle, active-idle)
			log.Printf("   📦 Internal Queue: %d messages", queueLen)
			log.Printf("   ✅ Processed: %d", processed)
			log.Printf("   ❌ Failed: %d", failed)
			log.Println("📊 ===============================================")
		}
	}
}

func (c *AutoScalingConsumer) Shutdown() {
	log.Println("🛑 Shutting down auto-scaling consumer...")

	c.cancel()

	log.Printf("⏳ Waiting for %d active workers to finish...", atomic.LoadInt32(&c.activeWorkers))
	c.workerWg.Wait()

	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}

	log.Println("✅ Graceful shutdown complete")
}

func downloadAndExtractPDF(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status: %d", resp.StatusCode)
	}

	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	reader := bytes.NewReader(pdfData)
	pdfReader, err := pdf.NewReader(reader, int64(len(pdfData)))
	if err != nil {
		return "", err
	}

	var text string
	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		pageText, _ := page.GetPlainText(nil)
		text += pageText + "\n"
	}

	return text, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	minWorkers := 2
	maxWorkers := 50

	if env := os.Getenv("MIN_WORKERS"); env != "" {
		fmt.Sscanf(env, "%d", &minWorkers)
	}
	if env := os.Getenv("MAX_WORKERS"); env != "" {
		fmt.Sscanf(env, "%d", &maxWorkers)
	}

	consumer, err := NewAutoScalingConsumer(
		"amqp://guest:guest@localhost:5672/",
		minWorkers,
		maxWorkers,
	)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}

	if err := consumer.Start(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("\n🔔 Interrupt received...")

	consumer.Shutdown()
}
