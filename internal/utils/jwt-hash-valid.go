package utils

import (
	"log"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

func CheckJWTHash(tokenString string) (int, error) {
	jwtToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("SECRET_KEY")), nil
	})
	if err != nil {
		log.Println("Не удалось распарсить токен")
		return 0, err
	}
	if err != nil || !jwtToken.Valid {
		return 0, err
	}
	claims := jwtToken.Claims.(jwt.MapClaims)
	id := int(claims["id"].(float64))
	return id, nil
}
