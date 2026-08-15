// Package utils contains shared infrastructure used by the services.
package utils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

var (
	// ErrProducerClosing means the service has started Kafka shutdown.
	ErrProducerClosing = errors.New("Kafka producer is closing")
	// ErrDeliveryUnknown means Kafka may still receive a queued message.
	ErrDeliveryUnknown = errors.New("Kafka delivery outcome unknown")
)

// KafkaProducer wraps Confluent's producer. Produce queues a message, while
// Publish waits for that message's delivery report before returning.
// See https://docs.confluent.io/kafka-clients/go/current/overview.html#asynchronous-writes.
type KafkaProducer struct {
	client *kafka.Producer

	// mu protects the shutdown and fatal-error state. Publish and Close must
	// see the same state, or a publish could start while Close is shutting down.
	mu       sync.Mutex
	closing  bool
	fatalErr error

	// inFlight keeps Close from closing the client while Publish is still
	// waiting for a delivery report.
	inFlight sync.WaitGroup

	// closeOnce makes repeated Close calls harmless.
	closeOnce sync.Once
	closeErr  error
}

// NewKafkaProducer creates a producer and starts one watcher for producer-wide
// events. Individual delivery reports are handled by Publish.
func NewKafkaProducer(brokers, clientID string) (*KafkaProducer, error) {
	client, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"client.id":         clientID,
		// Require all in-sync replicas to acknowledge the record and let
		// librdkafka keep producer retries safe and ordered.
		// See https://github.com/confluentinc/librdkafka/blob/master/CONFIGURATION.md.
		"acks":                                  "all",
		"enable.idempotence":                    true,
		"max.in.flight.requests.per.connection": 5,
		"retry.backoff.ms":                      300,
		"retry.backoff.max.ms":                  30000,
	})
	if err != nil {
		return nil, fmt.Errorf("create Kafka producer: %w", err)
	}

	producer := &KafkaProducer{
		client: client,
	}

	go producer.watchEvents()

	return producer, nil
}

func (p *KafkaProducer) watchEvents() {
	// Publish uses a private delivery channel for its message result. This
	// watcher remains necessary for producer-wide errors and for any caller
	// that produces without a private delivery channel.
	for event := range p.client.Events() {
		switch event := event.(type) {

		case *kafka.Message:
			// This is mostly for calls using Produce(..., nil).
			if event.TopicPartition.Error != nil {
				slog.Error(
					"Kafka delivery failed",
					"topic_partition", event.TopicPartition,
					"error", event.TopicPartition.Error,
				)
			} else {
				slog.Debug(
					"Kafka message delivered",
					"topic_partition", event.TopicPartition,
				)
			}

		case kafka.Error:
			if event.IsFatal() {
				// A fatal producer cannot safely accept new messages. Existing
				// publishes finish on their own delivery channel.
				p.mu.Lock()
				if p.fatalErr == nil {
					p.fatalErr = event
				}
				p.mu.Unlock()
				slog.Error(
					"fatal Kafka producer error",
					"error", event,
				)
			} else {
				slog.Warn(
					"Kafka producer event",
					"error", event,
				)
			}
		}
	}
}

// Publish queues one message and waits for Kafka to report its delivery.
// The context can stop the wait, but it cannot undo a message already queued.
func (p *KafkaProducer) Publish(ctx context.Context, topic string, key string, payload []byte) (kafka.TopicPartition, error) {
	p.mu.Lock()
	if p.closing {
		// Without this check, a request arriving during shutdown could call
		// Produce after the underlying client has been closed.
		p.mu.Unlock()
		return kafka.TopicPartition{}, ErrProducerClosing
	}
	if p.fatalErr != nil {
		// Fail fast rather than queueing work on a producer known to be broken.
		err := p.fatalErr
		p.mu.Unlock()
		return kafka.TopicPartition{}, fmt.Errorf("Kafka producer unavailable: %w", err)
	}
	// Add before releasing mu. Close takes the same lock before it waits, so
	// it cannot miss this publish between the check and the count update.
	p.inFlight.Add(1)
	p.mu.Unlock()
	// Every successful Add must have one matching Done, including errors and
	// context cancellation.
	defer p.inFlight.Done()

	delivery := make(chan kafka.Event, 1)

	err := p.client.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(key),
		Value: payload,
	}, delivery)

	if err != nil {
		return kafka.TopicPartition{},
			fmt.Errorf("enqueue Kafka message: %w", err)
	}
	select {
	case <-ctx.Done():
		// Important: delivery outcome is now unknown.
		// The queued message may still be delivered.
		return kafka.TopicPartition{},
			fmt.Errorf("%w: %w", ErrDeliveryUnknown, ctx.Err())
	case event := <-delivery:
		message, ok := event.(*kafka.Message)
		if !ok {
			return kafka.TopicPartition{},
				fmt.Errorf(
					"unexpected Kafka delivery event %T",
					event,
				)
		}

		if message.TopicPartition.Error != nil {
			return kafka.TopicPartition{},
				fmt.Errorf(
					"deliver Kafka message: %w",
					message.TopicPartition.Error,
				)
		}

		return message.TopicPartition, nil
	}

}

// Close flushes pending producer work and releases the client resources.
func (p *KafkaProducer) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		// Set this before waiting. New publishes now fail instead of extending
		// the shutdown indefinitely.
		p.closing = true
		p.mu.Unlock()

		// Existing publishes still need the client to deliver their result.
		// Closing first could leave them blocked or return an unknown outcome.
		p.inFlight.Wait()
		remaining := p.client.Flush(10_000)
		p.client.Close()

		if remaining > 0 {
			p.closeErr = fmt.Errorf(
				"%d Kafka messages remained undelivered during shutdown",
				remaining,
			)
		}
	})

	return p.closeErr
}

// KafkaConsumer wraps a consumer group subscription.

type KafkaConsumer struct {
	client *kafka.Consumer
}

// NewKafkaConsumer subscribes a consumer group to the supplied topics.
// Auto-commit is enabled for this first integration slice.
func NewKafkaConsumer(brokers string,
	groupID string,
	topics []string) (*KafkaConsumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"group.id":          groupID,
		// Use the earliest record only when this group has no committed offset.
		"auto.offset.reset": "earliest",
		// Kafka commits stored offsets in the background. StoreMessage below
		// decides when a handled message becomes eligible for that commit.
		// See https://github.com/confluentinc/librdkafka/blob/master/CONFIGURATION.md.
		"enable.auto.commit":       true,
		"enable.auto.offset.store": false,
	})
	if err != nil {
		return nil, fmt.Errorf("create Kafka consumer: %w", err)
	}

	if err := consumer.SubscribeTopics(topics, nil); err != nil {
		consumer.Close()
		return nil, fmt.Errorf("subscribe Kafka consumer: %w", err)
	}

	return &KafkaConsumer{client: consumer}, nil
}

// Consume polls Kafka and passes each message to handler.
// See https://docs.confluent.io/kafka-clients/go/current/overview.html#basic-usage.
func (c *KafkaConsumer) Consume(
	ctx context.Context,
	handler func(context.Context, *kafka.Message) error,
) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		event := c.client.Poll(100)

		switch event := event.(type) {
		case nil:
			continue

		case *kafka.Message:
			if err := handler(ctx, event); err != nil {
				return fmt.Errorf(
					"handle Kafka message: %w",
					err,
				)
			}

			// Store only after the handler succeeds. With auto-commit enabled,
			// librdkafka will commit this stored offset on its normal interval.
			// A crash before that commit can cause a duplicate, not a silent skip.
			if _, err := c.client.StoreMessage(event); err != nil {
				return fmt.Errorf(
					"store Kafka offset: %w",
					err,
				)
			}

		case kafka.Error:
			if event.IsFatal() {
				return fmt.Errorf(
					"fatal Kafka consumer error: %w",
					event,
				)
			}

			slog.Warn(
				"Kafka consumer event",
				"error", event,
			)
		}
	}
}

// Close leaves the consumer group and releases its network resources.
func (c *KafkaConsumer) Close() error {
	return c.client.Close()
}
