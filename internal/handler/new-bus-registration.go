package handler

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func RegisterBus(ctx *gin.Context, conn *sqlx.DB) {
	var bus models.Bus

	if err := ctx.ShouldBindJSON(&bus); err != nil {
		ctx.JSON(400, gin.H{"error": "Ошибка в модели Bus: " + err.Error()})
		return
	}

	tx, err := conn.Beginx()
	if err != nil {
		log.Println("Ошибка в транзакции", err)
		ctx.JSON(500, gin.H{"error": "Ошибка транзакции"})
		return
	}
	defer tx.Rollback()

	var lastBusID int
	queryBus := `
        INSERT INTO buses (bus_number, route_number, status) 
        VALUES ($1, $2, $3) 
        RETURNING id`

	err = tx.QueryRow(queryBus, bus.BusNumber, bus.RouteNumber, bus.Status).Scan(&lastBusID)
	if err != nil {
		log.Println("Ошибка добавления автобуса:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка сохранения автобуса: " + err.Error()})
		return
	}

	if len(bus.Devices) > 0 {
		for _, dev := range bus.Devices {
			queryDev := `
                INSERT INTO devices (rtsp_link, type, status, bus_id) 
                VALUES ($1, $2, $3, $4)`

			_, err = tx.Exec(queryDev, dev.RTSPlink, dev.Type, dev.Status, lastBusID)
			if err != nil {
				log.Println("Ошибка добавления девайса:", err)
				ctx.JSON(500, gin.H{"error": "Ошибка сохранения девайса: " + err.Error()})
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		ctx.JSON(500, gin.H{"error": "Не удалось подтвердить транзакцию"})
		return
	}

	ctx.JSON(200, gin.H{"status": "bus and devices registered", "id": lastBusID})
}
