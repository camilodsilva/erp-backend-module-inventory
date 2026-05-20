package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type CollaboratorClaims struct {
	Sub       string   `json:"sub"`
	Type      string   `json:"type"`
	CompanyID string   `json:"company_id"`
	TenantID  string   `json:"tenant_id"`
	Roles     []string `json:"roles"`
	Status    string   `json:"status"`
	jwt.RegisteredClaims
}

func ValidateCollaboratorToken(tokenString, secret string) (*CollaboratorClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CollaboratorClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*CollaboratorClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
