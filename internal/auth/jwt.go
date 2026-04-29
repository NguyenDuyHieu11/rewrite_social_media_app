package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID               string `json:"user_id"`
	Username             string `json:"username"`
	Role                 string `json:"role"`
	jwt.RegisteredClaims        // since i am embedding this type and this type already implemented those methods requrie by the root interface Claims, my own Claims interface now becomes the real Claims in jwt package. and the reason for both to coexist is go allow same name as long as they are in diff packages.
}

func GeneratingToken(userId string, userName string, role string, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID:   userId,
		Username: userName,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userId,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func ParseAccessToken(tokenToVerify string, secret []byte) (*Claims, error) {

	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims( // return type *jwt.Token
		tokenToVerify,
		claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secret, nil
		},
	)

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, err
		default:
			return nil, fmt.Errorf("%w: %v", jwt.ErrTokenInvalidClaims, err)
		}

	}

	return parsed.Claims.(*Claims), nil
}
