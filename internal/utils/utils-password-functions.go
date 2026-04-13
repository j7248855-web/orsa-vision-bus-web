package utils

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

// Шифрование пароля перед отправкой
func EncryptedPassword(password string) string {
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Fatalln("Не удалось зашифровать пароль:", err)
	}
	return string(hashPassword)
}

func CheckHashPassword(hashPassword string, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(password))
	if err != nil {
		log.Println("Хэши несовпадают")
		return err
	}
	return nil
}
