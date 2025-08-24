package utils

import (
	"golang.org/x/crypto/bcrypt"
)

func PwdGenerate(pwd string) (pwdHash string, err error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func PwdVerify(pwd string, pwdHash string) error {
	return bcrypt.CompareHashAndPassword([]byte(pwdHash), []byte(pwd))
}
