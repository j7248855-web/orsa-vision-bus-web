package auth

import (
	"fmt"
	"log"
	"orsavisionweb/internal/models"
	auxuliary "orsavisionweb/internal/utils/auxiliary"
	"orsavisionweb/internal/utils/jwtl"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func Login(ctx *gin.Context, conn *sqlx.DB) {
	var login models.Login

	err := ctx.ShouldBindJSON(&login)
	if err != nil {
		log.Println("Не удалось распарсить приходящие данные в структуру:", err)
	}
	var id int
	var hashPassword string
	fmt.Println(login)
	err = conn.QueryRow("SELECT id, password FROM users WHERE username=$1", login.Name).Scan(&id, &hashPassword)
	if err != nil {
		ctx.JSON(401, gin.H{"error": "Ошибка со стороны сервера"})
		log.Println("Не удалось достать пароль от пользователя:", err)
		return
	}
	if auxuliary.CheckHashPassword(hashPassword, login.Password) != nil {
		ctx.JSON(401, gin.H{"error": "Неверный пароль, пожалуйста перепроверьте"})
		return
	}
	//Формируем JWT и отправляем дальше
	token := jwtl.EncryptedToken(id)
	ctx.JSON(200, gin.H{"token": token})
}
