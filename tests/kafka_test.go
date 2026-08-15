//go:build integration

package tests

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/JustUzair/go-grpc-irctc-backend/utils"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func TestKafkaProducerConsumerIntegration(t *testing.T) {
	// Use a unique topic so the test can run without touching application topics.
	// Cleanup removes it when the test finishes.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	config, err := env.Load()
	if err != nil {
		t.Fatalf("load test configuration: %v", err)
	}

	brokers := config.KafkaBrokers
	if brokers == "" {
		brokers = "localhost:9093"
	}

	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		t.Fatalf("create Kafka admin client: %v", err)
	}
	t.Cleanup(func() { admin.Close() })

	topic := fmt.Sprintf("test.user.email.otp.%d", time.Now().UnixNano())
	_, err = admin.CreateTopics(ctx, []kafka.TopicSpecification{
		{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	})
	if err != nil {
		t.Fatalf("create Kafka test topic: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := admin.DeleteTopics(cleanupCtx, []string{topic}); err != nil {
			t.Errorf("delete Kafka test topic: %v", err)
		}
	})

	producer, err := utils.NewKafkaProducer(brokers, "integration-test-producer")
	if err != nil {
		t.Fatalf("create Kafka producer: %v", err)
	}
	t.Cleanup(func() {
		if err := producer.Close(); err != nil {
			t.Errorf("close Kafka producer: %v", err)
		}
	})

	consumer, err := utils.NewKafkaConsumer(
		brokers,
		fmt.Sprintf("integration-test-group-%d", time.Now().UnixNano()),
		[]string{topic},
	)
	if err != nil {
		t.Fatalf("create Kafka consumer: %v", err)
	}

	consumeCtx, stopConsuming := context.WithCancel(ctx)
	consumeDone := make(chan error, 1)
	received := make(chan string, 1)

	go func() {
		consumeDone <- consumer.Consume(consumeCtx, func(_ context.Context, message *kafka.Message) error {
			received <- string(message.Value)
			return nil
		})
	}()

	t.Cleanup(func() {
		stopConsuming()
		if err := <-consumeDone; err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("consume Kafka test message: %v", err)
		}
		if err := consumer.Close(); err != nil {
			t.Errorf("close Kafka consumer: %v", err)
		}
	})

	const payload = `{"event":"otp-requested"}`
	if _, err := producer.Publish(ctx, topic, "test-user", []byte(payload)); err != nil {
		t.Fatalf("publish Kafka test message: %v", err)
	}

	select {
	case value := <-received:
		if value != payload {
			t.Fatalf("received payload %q, want %q", value, payload)
		}
	case <-ctx.Done():
		t.Fatalf("wait for Kafka test message: %v", ctx.Err())
	}
}
