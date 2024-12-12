package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var jwtKey = []byte("your-secret-key")

type Claims struct {
	Email     string `json:"email"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(email string) (string, error) {
	// Thời gian hết hạn của accessToken (ví dụ: 15 phút)
	expirationTime := time.Now().Add(15 * time.Minute)
	claims := &Claims{
		Email:     email,
		TokenType: "access_token",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "chat-app",
		},
	}

	// Tạo token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Ký token
	accessToken, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

// Tạo refreshToken
func GenerateRefreshToken(email string) (string, error) {
	// Thời gian hết hạn của refreshToken (ví dụ: 7 ngày)
	expirationTime := time.Now().Add(7 * 24 * time.Hour)
	claims := &Claims{
		Email:     email,
		TokenType: "refresh_token",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "chat-app",
		},
	}

	// Tạo token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Ký token
	refreshToken, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return refreshToken, nil
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}

	// Kiểm tra token
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func CheckTokenType(tokenString string) string {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return ""
	}

	return claims.TokenType
}
