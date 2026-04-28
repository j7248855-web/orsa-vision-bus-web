package handler

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func ReturnNewUser(ctx *gin.Context, conn *sqlx.DB) {
	var users []models.CreateUsers

	err := conn.Select(&users, "SELECT username, full_name, permissions FROM users ORDER BY created_at DESC")

	if err != nil {
		log.Println("Ошибка при получении пользователей:", err)
		ctx.JSON(500, gin.H{"error": "Не удалось получить список пользователей"})
		return
	}
	if users == nil {
		users = []models.CreateUsers{}
	}

	ctx.JSON(200, users)
}
