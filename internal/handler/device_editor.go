package handler

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func DeviceEditor(ctx *gin.Context, conn *sqlx.DB) {
	var devices models.Device
	devicesId := ctx.Param("device_id")
	ctx.ShouldBindBodyWithJSON(&devices)
	_, err := conn.Exec("UPDATE devices SET rtsp_link=$1, type=$2, status=$3 WHERE id=$4", devices.RTSPlink, devices.Type, devices.Status, devicesId)
	if err != nil {
		log.Println("Не удалось обновить таблицу по причине:", err)
		ctx.JSON(500, gin.H{"status": "Ошибка на сервере"})
		return
	}
	ctx.JSON(200, gin.H{"status": "success"})
}
