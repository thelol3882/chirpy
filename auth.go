package main

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/thelol3882/chirpy/internal/auth"
)

func (cfg *apiConfig) authenticatedUserID(r *http.Request) (uuid.UUID, error) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		return uuid.Nil, err
	}

	return auth.ValidateJWT(token, cfg.secretKey)
}
