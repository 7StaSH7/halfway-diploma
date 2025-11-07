package utils

import (
	"errors"
	"strings"
)

func ValidateLuhn(number string) error {
	cleanNumber := strings.ReplaceAll(number, " ", "")

	if !isNumeric(cleanNumber) {
		return errors.New("number contains non-digit characters")
	}

	digits := []byte(cleanNumber)
	length := len(digits)

	if length < 2 {
		return errors.New("number must have at least 2 digits")
	}

	if length > 20 {
		return errors.New("number too long")
	}

	sum := 0
	parity := length % 2

	for i := range length {
		digit := int(digits[i] - '0')

		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	if sum%10 != 0 {
		return errors.New("number failed Luhn validation")
	}

	return nil
}

func isNumeric(s string) bool {
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return len(s) > 0
}
