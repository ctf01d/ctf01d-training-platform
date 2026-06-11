package auth

import (
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

type PasswordCheckerImpl struct{}

func (p *PasswordCheckerImpl) CheckPassword(hash, plain string) bool {
	return CheckPassword(hash, plain)
}
