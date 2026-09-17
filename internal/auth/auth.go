package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func getAuthToken(headers http.Header, scheme string) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no authorization header provided")
	}

	fields := strings.Fields(authHeader)

	if len(fields) != 2 || !strings.EqualFold(fields[0], scheme) {
		return "", errors.New("invalid token")
	}

	return fields[1], nil
}

func GetBearerToken(headers http.Header) (string, error) {
	return getAuthToken(headers, "Bearer")
}

func GetAPIKey(headers http.Header) (string, error) {
	return getAuthToken(headers, "ApiKey")
}
