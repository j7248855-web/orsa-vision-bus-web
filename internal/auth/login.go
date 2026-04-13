package auth

import (
	"log"
	"orsavisionweb/internal/models"
	"orsavisionweb/internal/utils"

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
	err = conn.QueryRow("SELECT id, password FROM users WHERE username=$1", login.Name).Scan(&id, &hashPassword)
	if err != nil {
		ctx.JSON(401, gin.H{"error": "Ошибка со стороны сервера"})
		log.Println("Не удалось достать пароль от пользователя:", err)
		return
	}
	if utils.CheckHashPassword(hashPassword, login.Password) != nil {
		ctx.JSON(401, gin.H{"error": "Неверный пароль, пожалуйста перепроверьте"})
		return
	}
	//Формируем JWT и отправляем дальше
	token := utils.EncryptedToken(id)
	ctx.JSON(200, gin.H{"token": token})
}
