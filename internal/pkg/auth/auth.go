package auth

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

type Auth struct {
	Key    []byte
	Method jwt.SigningMethod
}

type User struct {
	ID   uint64 `json:"id"`
	Name string `json:"username"`
}

type Claims struct {
	jwt.StandardClaims
	User User `json:"user"`
}

func NewAuth() *Auth {
	return &Auth{
		Key:    []byte("hello"),
		Method: jwt.GetSigningMethod("HS256"),
	}
}

func (a *Auth) GetSignedToken(user User, issuedAt time.Time, ttl time.Duration) (string, string, error) {
	id := uuid.New().String()
	claims := Claims{
		User: struct {
			ID   uint64 `json:"id"`
			Name string `json:"username"`
		}{
			ID:   user.ID,
			Name: user.Name,
		},
		StandardClaims: jwt.StandardClaims{
			Id:        id,
			IssuedAt:  issuedAt.Unix(),
			ExpiresAt: issuedAt.Add(ttl).Unix(),
		},
	}
	token := jwt.NewWithClaims(a.Method, claims)

	str, err := token.SignedString(a.Key)
	if err != nil {
		return "", "", fmt.Errorf("Could not sign token %v", err)
	}
	return str, id, nil
}

func (a *Auth) ExtractClaims(tokenStr string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unsupported signing method %v", token.Header["alg"])
		}
		return a.Key, nil
	})

	if !token.Valid || err != nil {
		return Claims{}, fmt.Errorf("Could not parse token %v", err)
	}
	return claims, nil
}
