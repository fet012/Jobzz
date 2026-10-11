package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword takes a plaintext password and returns a bcrypt hash.
// The hash is a self-contained string that includes the salt and cost factor.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword compares a plaintext password to a stored hash.
// Returns nil on match, an error otherwise.
func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

