package handler

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func DeviceRemove(ctx *gin.Context, conn *sqlx.DB) {
	deviceID := ctx.Param("device_id")
	_, err := conn.Exec("DELETE FROM devices WHERE id = $1", deviceID)
	if err != nil {
		log.Println("Ошибка при удалении девайса:", err)
		ctx.JSON(500, gin.H{"status": "Ошибка на стороне сервера"})
	}
}
