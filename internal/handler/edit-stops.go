package handler

import (
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func EditBusStops(ctx *gin.Context, conn *sqlx.DB) {
	var stop models.Stop

	// if _, exists := ctx.Get("id"); !exists {
	// 	ctx.JSON(401, gin.H{"status": "Вы не авторизованы"})
	// 	return
	// }

	if err := ctx.ShouldBindJSON(&stop); err != nil {
		ctx.JSON(400, gin.H{"error": "Ошибка в формате JSON"})
		return
	}

	lat := stop.Position[0]
	lng := stop.Position[1]

	query := `
        UPDATE stops 
        SET name=$1, lat=$2, lng=$3, radius=$4, type=$5, azimuth=$6, city=$7 
        WHERE id=$8`

	_, err := conn.Exec(query,
		stop.Name,
		lat, // Берем из вытащенного
		lng, // Берем из вытащенного
		stop.Radius,
		stop.Type,
		stop.Azimuth,
		stop.City,
		stop.ID, // ID из JSON
	)

	if err != nil {
		ctx.JSON(500, gin.H{"error": "Ошибка БД: " + err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"status": "success"})
}
