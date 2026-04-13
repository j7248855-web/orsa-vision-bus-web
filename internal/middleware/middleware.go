package middleware

import (
	"log"
	"orsavisionweb/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func MiddleWareAuth(ctx *gin.Context) {
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		ctx.JSON(401, gin.H{"Нет токена": "Нет токена"})
		return
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		log.Println("Не удалось найти значение Bearer")
		ctx.AbortWithStatusJSON(401, gin.H{"status": "Не удалось проверить токен, доступ закрыт"})
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	id, err := utils.CheckJWTHash(token)
	if err != nil {
		ctx.AbortWithStatusJSON(401, gin.H{"status": "Не удалось проверить токен, доступ закрыт"})
		return
	}
	ctx.Set("uuid", id)
	ctx.Next()
}
