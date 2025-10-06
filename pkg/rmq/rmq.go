package rmq

import (
	"context"
	"errors"
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
		_ = conn.Close()
		return nil, err
	}

	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &Publisher{conn: conn, ch: ch, queue: queue}, nil
}

func (p *Publisher) Close() error {
	var cerr error
	if p.ch != nil {
		if err := p.ch.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			cerr = err
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) && cerr == nil {
			cerr = err
		}
	}
	return cerr
}

func (p *Publisher) PublishJSON(ctx context.Context, body []byte) error {
	return p.PublishJSONWithHeaders(ctx, body, nil)
}

func (p *Publisher) PublishJSONWithHeaders(ctx context.Context, body []byte, headers amqp.Table) error {
	return p.ch.PublishWithContext(
		ctx,
		"", p.queue, // exchange, key
		false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
			Headers:      headers,
		},
	)
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
		_ = conn.Close()
		return nil, err
	}

	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	if err := ch.Qos(10, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &Consumer{conn: conn, Ch: ch, Queue: queue}, nil
}

func (c *Consumer) Consume() (<-chan amqp.Delivery, error) {
	return c.Ch.Consume(c.Queue, "", false, false, false, false, nil)
}

func (c *Consumer) Close() error {
	var cerr error
	if c.Ch != nil {
		if err := c.Ch.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			cerr = err
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) && cerr == nil {
			cerr = err
		}
	}
	return cerr
}
