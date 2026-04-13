package handler

import (
	"fmt"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func FullBusStation(ctx *gin.Context, conn *sqlx.DB) {

	city := ctx.Param("city")
	var dbStops []models.Stop
	query := `SELECT id, name, lat, lng, radius, type, azimuth, city FROM stops WHERE city = $1`
	err := conn.Select(&dbStops, query, city)
	if err != nil {
		fmt.Println("Ошибка Select:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка получения данных"})
		return
	}

	for i := range dbStops {
		dbStops[i].Position = [2]float64{dbStops[i].Lon, dbStops[i].Lat}
	}
	if dbStops == nil {
		dbStops = []models.Stop{}
	}

	ctx.JSON(200, dbStops)
}
