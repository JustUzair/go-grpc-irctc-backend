package constants

var TOPIC = struct {
	OTP_EMAIL     string
	WELCOME_EMAIL string
	BOOKING_EMAIL string
	PAYMENT_EMAIL string
}{
	OTP_EMAIL:     "notification.otp-email",
	WELCOME_EMAIL: "notification.welcome-email",
	BOOKING_EMAIL: "notification.booking-email",
	PAYMENT_EMAIL: "notification.payment-email",
}

func AllTopics() []string {
	return []string{
		TOPIC.OTP_EMAIL,
		TOPIC.WELCOME_EMAIL,
		TOPIC.BOOKING_EMAIL,
		TOPIC.PAYMENT_EMAIL,
	}
}
