package custom_types

type OTPEmailKafkaPacket struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	OTP        string `json:"otp"`
	TTLMinutes int    `json:"ttl_minutes"`
}

type VerifyEmailKafkaPacket struct {
	Email     string `json:"email"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
}
