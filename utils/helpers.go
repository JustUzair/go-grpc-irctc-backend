package utils

import "net/mail"

func IsEmailValid(email string) bool {
	addr, err := mail.ParseAddress(email)
	// Checks that parsing succeeded and matches the input exactly (no extra names)
	return err == nil && addr.Address == email
}
