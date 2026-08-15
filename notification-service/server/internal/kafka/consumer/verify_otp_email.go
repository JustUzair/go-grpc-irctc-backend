package notification_consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JustUzair/go-grpc-irctc-backend/utils"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/mailer"
	custom_types "github.com/JustUzair/go-grpc-irctc-backend/utils/types"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// HandleVerifyOTPEmail returns the Kafka callback for welcoming new email events.
func HandleVerifyOTPEmail(mailService mailer.Mailer) func(context.Context, *kafka.Message) error {
	return func(ctx context.Context, message *kafka.Message) error {
		var packet custom_types.VerifyEmailKafkaPacket
		if err := json.Unmarshal(message.Value, &packet); err != nil {
			return fmt.Errorf("decode OTP email event: %w", err)
		}

		if !utils.IsEmailValid(packet.Email) {
			return fmt.Errorf("invalid welcome email event")
		}

		if _, err := mailService.SendEmail(ctx, mailer.VerifyOTP, mailer.EmailParams{
			ToEmailAddress: packet.Email,
			TemplateData: mailer.VerifyOTPTemplateData{
				Name: packet.FirstName + " " + packet.LastName,
			},
		}); err != nil {
			return fmt.Errorf("welcome user email: %w", err)
		}

		slog.Info(
			"welcome notification delivered",
			"topic", *message.TopicPartition.Topic,
			"partition", message.TopicPartition.Partition,
			"offset", message.TopicPartition.Offset,
		)

		return nil
	}
}
