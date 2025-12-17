package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/amrrdev/trawl/services/indexing/internal/types"
	"github.com/amrrdev/trawl/services/shared/queue"
)

type Producer struct {
	client    *queue.RabbitMQ
	queueName string
}

func NewProducer(client *queue.RabbitMQ, queueName string) (*Producer, error) {
	dlqArgs := map[string]interface{}{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": queueName + "_dlq",
	}

	if err := client.DeclareQueue(queueName, true, dlqArgs); err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := client.DeclareQueue(queueName+"_dlq", true, nil); err != nil {
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	log.Printf("✓ Queues declared: %s, %s_dlq", queueName, queueName)

	return &Producer{
		client:    client,
		queueName: queueName,
	}, nil
}

func (p *Producer) PublishIndexingJob(ctx context.Context, job *types.IndexingJob) error {
	fmt.Println("inside pusshing")
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := p.client.Publish(p.queueName, data); err != nil {
		return fmt.Errorf("failed to publish job: %w", err)
	}

	log.Printf("✓ Job published: %s (DocID: %s)", job.JobID, job.Payload.DocID)
	return nil
}
