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

// HandleSendOTPEmail returns the Kafka callback for OTP email events.
func HandleSendOTPEmail(mailService mailer.Mailer) func(context.Context, *kafka.Message) error {
	return func(ctx context.Context, message *kafka.Message) error {
		var packet custom_types.OTPEmailKafkaPacket
		if err := json.Unmarshal(message.Value, &packet); err != nil {
			return fmt.Errorf("decode OTP email event: %w", err)
		}

		if !utils.IsEmailValid(packet.Email) ||
			packet.OTP == "" ||
			packet.TTLMinutes <= 0 {
			return fmt.Errorf("invalid OTP email event")
		}

		if _, err := mailService.SendEmail(ctx, mailer.SendOTP, mailer.EmailParams{
			ToEmailAddress: packet.Email,
			TemplateData: mailer.SendOTPTemplateData{
				Name:             packet.Name,
				OTP:              packet.OTP,
				ExpiresInMinutes: packet.TTLMinutes,
			},
		}); err != nil {
			return fmt.Errorf("send OTP email: %w", err)
		}
		slog.Info(
			"notification delivered",
			"topic", *message.TopicPartition.Topic,
			"partition", message.TopicPartition.Partition,
			"offset", message.TopicPartition.Offset,
		)

		return nil
	}
}
