package handler

import (
	"log"
	"orsavisionweb/internal/models"
	"orsavisionweb/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func CreateNewUser(ctx *gin.Context, conn *sqlx.DB) {
	var users models.CreateUsers
	if err := ctx.ShouldBindJSON(&users); err != nil {
		ctx.JSON(401, gin.H{"error": "Неверные данные"})
		log.Println("Ошибка:", err)
		return
	}
	hashedPassword := utils.EncryptedPassword(users.Password)
	_, err := conn.Exec("INSERT INTO users (username, password, full_name, created_at) VALUES ($1, $2, $3, $4)", users.Login, hashedPassword, users.FullName, time.Now())
	if err != nil {
		ctx.JSON(409, gin.H{"error": "Пользователь с таким никнеймом уже существует, придумайте другой"})
		return
	}
	ctx.JSON(200, gin.H{"status": "success"})
}
