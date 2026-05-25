package devices

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Редактирование девайса
func DeviceEditor(ctx *gin.Context, conn *sqlx.DB) {
	var devices models.Device
	devicesId := ctx.Param("device_id")
	ctx.ShouldBindJSON(&devices)
	result, err := conn.ExecContext(ctx, "UPDATE devices SET device_ip=$1, type=$2, status=$3 WHERE id=$4", devices.DeviceIP, devices.Type, devices.Status, devicesId)
	if err != nil {
		log.Println("Не удалось обновить таблицу по причине:", err)
		ctx.JSON(500, gin.H{"status": "Ошибка на сервере"})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		ctx.JSON(404, gin.H{"error": "Девайс с таким ID не найден"})
		return
	}
	ctx.JSON(200, gin.H{"status": "success"})
}

// Удаление девайса
func DeviceRemove(ctx *gin.Context, conn *sqlx.DB) {
	deviceID := ctx.Param("device_id")
	result, err := conn.ExecContext(ctx, "DELETE FROM devices WHERE id = $1", deviceID)
	if err != nil {
		log.Println("Ошибка при удалении девайса:", err)
		ctx.JSON(500, gin.H{"status": "Ошибка на стороне сервера"})
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		ctx.JSON(404, gin.H{"error": "Девайс с таким ID не найден"})
		return
	}
	ctx.JSON(200, gin.H{"status": "success"})
}
