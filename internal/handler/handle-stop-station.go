package handler

import (
	"fmt"
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func HandleRouteStops(ctx *gin.Context, conn *sqlx.DB) {
	var stop models.Stop

	if err := ctx.ShouldBindJSON(&stop); err != nil {
		ctx.JSON(400, gin.H{"error": "Ошибка в модели Stop: " + err.Error()})
		log.Println("Ошибка парсинга:", err)
		return
	}

	_, err := conn.Exec(`
        INSERT INTO stops (name, lat, lng, radius, type, city, azimuth) 
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (id) DO UPDATE SET 
            name = EXCLUDED.name, 
            lat = EXCLUDED.lat, 
            lng = EXCLUDED.lng,
            azimuth = EXCLUDED.azimuth`,
		stop.Name, stop.Position[1], stop.Position[0], stop.Radius, stop.Type, stop.City, stop.Azimuth)

	if err != nil {
		ctx.JSON(500, gin.H{"error": "Ошибка на стороне сервера"})
		fmt.Println("Не удалось распарсить ошибка:", err)
		return
	}

	ctx.JSON(200, gin.H{"status": "stop updated"})
}
