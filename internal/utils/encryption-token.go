package utils

import (
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func EncryptedToken(userID int) string {
	var claims = make(jwt.MapClaims)
	claims["id"] = userID
	claims["exp"] = time.Now().Add(time.Hour * 48).Unix() //Устанавливаем время жизни токена
	//Создаём сам токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	key := []byte(os.Getenv("SECRET_KEY"))
	tokenString, err := token.SignedString(key)
	if err != nil {
		log.Fatalln("Не удалось создать токен:", err)
	}
	return tokenString
}
