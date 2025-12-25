package queue

import (
	"fmt"

	"github.com/amrrdev/trawl/services/shared/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	client    *queue.RabbitMQ
	queueName string
	dlqName   string
}

func NewConsumer(client *queue.RabbitMQ, queueName, dqlName string) (*Consumer, error) {
	consumer := &Consumer{
		client:    client,
		queueName: queueName,
		dlqName:   dqlName,
	}

	if err := consumer.declareQueue(); err != nil {
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	if err := consumer.declareDLQ(); err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := client.Channel.Qos(10, 0, false); err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return consumer, nil
}

func (c *Consumer) declareDLQ() error {
	_, err := c.client.Channel.QueueDeclare(
		c.dlqName,
		true,
		false,
		false,
		false,
		nil,
	)

	return err
}

func (c *Consumer) declareQueue() error {
	args := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": c.dlqName,
	}

	_, err := c.client.Channel.QueueDeclare(c.queueName,
		true,
		false,
		false,
		false,
		args,
	)
	return err
}

func (c *Consumer) Consume() (<-chan amqp.Delivery, error) {
	consumed, err := c.client.Channel.Consume(c.queueName, "indexing-worker", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to consume from %s queue: %s", c.queueName, err)
	}

	return consumed, nil
}

func (c *Consumer) Publish(data []byte, headers map[string]interface{}) error {
	err := c.client.Channel.Publish("", c.queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         data,
		Headers:      headers,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		return fmt.Errorf("failed to publish message in queue: %s", err)
	}
	return nil
}

func (c *Consumer) Close() error {
	return c.client.Close()
}
