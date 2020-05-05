package main

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func isValidEmailAddress(emailAddress string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return re.MatchString(emailAddress)
}

func generateUUID() (string, error) {
	unix32bits := uint32(time.Now().UTC().Unix())

	buff := make([]byte, 12)

	_, err := rand.Read(buff)

	token := fmt.Sprintf("%x%x%x%x%x", unix32bits, buff[0:2], buff[2:4], buff[4:6], buff[6:])

	return token, err
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	bytes, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), nil
}

func GenerateRandomStringURLSafe(n int) (string, error) {
	b, err := GenerateRandomBytes(n)
	return base64.URLEncoding.EncodeToString(b), err
}

func generateActivationToken(len int) (string, error) {
	uuid, err := generateUUID()
	if err != nil {
		println(err.Error())
		return "", err
	}

	rsus, err := GenerateRandomStringURLSafe(len)
	if err != nil {
		println(err.Error())
		return "", err
	}

	return uuid + rsus, nil

}
