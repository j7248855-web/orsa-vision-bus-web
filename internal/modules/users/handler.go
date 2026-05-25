package users

import (
	"log"
	"orsavisionweb/internal/models"
	auxuliary "orsavisionweb/internal/utils/auxiliary"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Создание нового пользователя
func CreateNewUser(ctx *gin.Context, conn *sqlx.DB) {
	var users models.CreateUsers
	if err := ctx.ShouldBindJSON(&users); err != nil {
		ctx.JSON(400, gin.H{"error": "Неверно заполненные поля данных", "details": err.Error()})
		log.Println("Ошибка:", err)
		return
	}
	hashedPassword := auxuliary.EncryptedPassword(users.Password)
	_, err := conn.ExecContext(ctx, "INSERT INTO users (username, password, full_name, permissions, created_at) VALUES ($1, $2, $3, $4, $5)", users.Login, hashedPassword, users.FullName, users.Permissions, time.Now().UTC())
	if err != nil {
		ctx.JSON(409, gin.H{"error": "Не удалось создать нового пользователя", "details": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"status": "success"})
}

// Возвращение всех новых пользователей
func ReturnNewUser(ctx *gin.Context, conn *sqlx.DB) {
	var users []models.CreateUsers

	err := conn.Select(&users, "SELECT id, username, full_name, permissions FROM users ORDER BY created_at DESC")

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

// Удаление пользователя
func RemoveUser(ctx *gin.Context, conn *sqlx.DB) {
	userIDStr := ctx.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Неверный формат ID пользователя"})
		return
	}
	//Проверка чтобы юзер не мог сам себя удалить
	currentID, _ := ctx.Get("uuid")
	if currentID.(int) == userID {
		ctx.JSON(500, gin.H{"error": "Вы не можете удалить собственную учётную запись"})
	}

	query := `DELETE FROM users WHERE id = $1`
	result, err := conn.ExecContext(ctx, query, userID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Ошибка при удалении из базы: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		ctx.JSON(404, gin.H{"error": "Пользователь с таким ID не найден"})
		return
	}

	ctx.JSON(200, gin.H{"status": "success"})
}

// Редактирование пользователя
func EditUser(ctx *gin.Context, conn *sqlx.DB) {
	var updatedUser models.CreateUsers

	if err := ctx.ShouldBindJSON(&updatedUser); err != nil {
		ctx.JSON(400, gin.H{"error": "Непредвиденная ошибка", "details": err.Error()})
		return
	}
	//Заново хэшируем пароль
	hashPassword := auxuliary.EncryptedPassword(updatedUser.Password)
	query := `
        UPDATE users 
        SET username = $1, password = $2, full_name = $3, permissions = $4 
        WHERE id = $5`

	result, err := conn.ExecContext(ctx, query,
		updatedUser.Login,
		hashPassword,
		updatedUser.FullName,
		updatedUser.Permissions,
		updatedUser.ID,
	)

	if err != nil {
		ctx.JSON(500, gin.H{"error": "Ошибка в БД: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		ctx.JSON(404, gin.H{"error": "Пользователь с таким ID не найден"})
		return
	}

	ctx.JSON(200, gin.H{"status": "success"})
}
