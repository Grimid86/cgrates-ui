// Package security provides hashing, password verification, and input validation.
package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string, cost int) (string, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

// VerifyPassword checks if password matches hash
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomToken generates a cryptographically secure random token
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateEmail checks email format
func ValidateEmail(email string) bool {
	re := regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)
	return re.MatchString(email)
}

// ValidateMSISDN checks MSISDN format (digits only)
func ValidateMSISDN(msisdn string) bool {
	re := regexp.MustCompile(`^[0-9]+$`)
	return re.MatchString(msisdn) && len(msisdn) >= 7 && len(msisdn) <= 15
}

// ValidateIMSI checks IMSI format
func ValidateIMSI(imsi string) bool {
	re := regexp.MustCompile(`^[0-9]+$`)
	return re.MatchString(imsi) && len(imsi) == 15
}

// ValidateHexColor checks CSS hex color format
func ValidateHexColor(color string) bool {
	re := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	return re.MatchString(color)
}
