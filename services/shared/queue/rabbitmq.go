package queue

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %s", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a RabbitMQ channel: %s", err)
	}

	return &RabbitMQ{
		Conn:    conn,
		Channel: channel,
	}, nil
}

func (r *RabbitMQ) DeclareQueue(name string, durable bool, args amqp.Table) error {
	_, err := r.Channel.QueueDeclare(name, durable, false, false, false, args)
	if err != nil {
		return fmt.Errorf("failed to declare a %s queue: %s", name, err)
	}
	return nil
}

func (r *RabbitMQ) Publish(queueName string, data []byte) error {
	err := r.Channel.Publish("", queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         data,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		return fmt.Errorf("failed to publish message in queue: %s", err)
	}
	return nil
}

func (r *RabbitMQ) Consume(queueName string) (<-chan amqp.Delivery, error) {
	delivery, err := r.Channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to consume from %s queue: %s", queueName, err)
	}

	return delivery, nil
}

func (r *RabbitMQ) Close() error {
	if err := r.Channel.Close(); err != nil {
		return fmt.Errorf("failed to close channel: %w", err)
	}
	if err := r.Conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}
	return nil
}
