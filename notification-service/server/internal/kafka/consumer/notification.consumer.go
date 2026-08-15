package notification_consumer

import (
	"context"
	"fmt"

	"github.com/JustUzair/go-grpc-irctc-backend/utils/constants"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/mailer"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// ConsumeNotification routes each subscribed Kafka topic to its focused
// handler. Add a handler here when a new notification type is implemented.
func ConsumeNotification(mailService mailer.Mailer) func(context.Context, *kafka.Message) error {

	return func(ctx context.Context, message *kafka.Message) error {
		if message == nil || message.TopicPartition.Topic == nil {
			return fmt.Errorf("Kafka message has no topic")
		}
		switch *message.TopicPartition.Topic {
		case constants.TOPIC.OTP_EMAIL:
			return HandleSendOTPEmail(mailService)(ctx, message)
		case constants.TOPIC.WELCOME_EMAIL:
			return HandleVerifyOTPEmail(mailService)(ctx, message)
		case constants.TOPIC.BOOKING_EMAIL,
			constants.TOPIC.PAYMENT_EMAIL:
			return fmt.Errorf("notification topic not supported yet: %s", *message.TopicPartition.Topic)
		default:
			return fmt.Errorf("unsupported notification topic: %s", *message.TopicPartition.Topic)
		}
	}
}
