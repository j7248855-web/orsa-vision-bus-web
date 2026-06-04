package auth

import (
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
	var dbUser struct {
		ID       int    `db:"id"`
		Password string `db:"password"`
	}
	err = conn.GetContext(ctx, &dbUser, "SELECT id, password FROM users WHERE username = $1 LIMIT 1", login.Name)
	if err != nil {
		ctx.JSON(401, gin.H{"error": "Пользователь не найден или ошибка сервера"})
		log.Println("Не удалось достать пользователя из БД:", err)
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
