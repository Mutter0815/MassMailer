package rmq

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	conn  *amqp.Connection
	ch    *amqp.Channel
	queue string
}

func NewPublisher(url, queue string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err

	}

	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	return &Publisher{conn: conn, ch: ch, queue: queue}, nil
}

func (p *Publisher) Close() error {
	_ = p.ch.Close()
	return p.conn.Close()
}

func (p *Publisher) PublishJSON(ctx context.Context, body []byte) error {
	return p.ch.PublishWithContext(ctx,
		"", p.queue, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		})
}

type Consumer struct {
	conn  *amqp.Connection
	Ch    *amqp.Channel
	Queue string
}

func NewConsumer(url, queue string) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	_ = ch.Qos(10, 0, false)
	return &Consumer{conn: conn, Ch: ch, Queue: queue}, nil
}

func (c *Consumer) Consume() (<-chan amqp.Delivery, error) {
	return c.Ch.Consume(c.Queue, "", false, false, false, false, nil)
}

func (c *Consumer) Close() error {
	_ = c.Ch.Close()
	return c.conn.Close()
}
