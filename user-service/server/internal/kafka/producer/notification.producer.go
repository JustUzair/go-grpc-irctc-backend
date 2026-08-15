package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JustUzair/go-grpc-irctc-backend/utils"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/constants"
	custom_types "github.com/JustUzair/go-grpc-irctc-backend/utils/types"
)

// SendOtpEmail publishes the data required by notification-service to render
// and send an OTP email.
func SendOtpEmail(ctx context.Context, kafkaClient *utils.KafkaProducer, email, name, otp string, ttlMinutes int) error {
	packet := custom_types.OTPEmailKafkaPacket{
		Name:       name,
		Email:      email,
		OTP:        otp,
		TTLMinutes: ttlMinutes,
	}

	payload, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("send otp email payload marshal error: %w", err)
	}
	key := fmt.Sprintf("otp-%s", email)
	return handleSendPayload(ctx, kafkaClient, constants.TOPIC.OTP_EMAIL, key, payload)
}

func VerifyOtpEmail(ctx context.Context, kafkaClient *utils.KafkaProducer, email, firstname, lastname string) error {
	packet := custom_types.VerifyEmailKafkaPacket{
		Email:     email,
		FirstName: firstname,
		LastName:  lastname,
	}
	payload, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("verify otp email payload marshal error: %w", err)
	}
	key := fmt.Sprintf("welcome-%s", email)
	return handleSendPayload(ctx, kafkaClient, constants.TOPIC.WELCOME_EMAIL, key, payload)

}

func handleSendPayload(ctx context.Context, kafkaClient *utils.KafkaProducer, topic string, key string, payload []byte) error {
	partition, err := kafkaClient.Publish(
		ctx,
		topic,
		key,
		payload,
	)
	if err != nil {
		return fmt.Errorf("kafka producer: %w", err)
	}

	slog.Info(
		"Kafka message sent",
		"topic", topic,
		"partition", partition.Partition,
		"offset", partition.Offset,
	)
	return nil
}
