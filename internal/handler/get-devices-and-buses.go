package handler

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func GetBuses(ctx *gin.Context, conn *sqlx.DB) {
	var buses []models.Bus

	err := conn.Select(&buses, "SELECT id, bus_number, route_number, status FROM buses")
	if err != nil {
		log.Println("Ошибка получения автобусов:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка базы данных при поиске автобусов"})
		return
	}

	for i := range buses {
		var devices []models.Device

		err := conn.Select(&devices, `
			SELECT id, rtsp_link, type, status 
			FROM devices 
			WHERE bus_id = $1`, buses[i].ID)

		if err != nil {
			log.Printf("Ошибка получения девайсов для автобуса %s: %v", buses[i].ID, err)
			// Не прерываем всё, просто оставляем пустой слайс для этого автобуса
			buses[i].Devices = []models.Device{}
			continue
		}

		if devices == nil {
			devices = []models.Device{}
		}

		buses[i].Devices = devices
	}

	// Если автобусов вообще нет в базе
	if buses == nil {
		buses = []models.Bus{}
	}

	ctx.JSON(200, buses)
}
