package stops

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func HandleRouteStops(ctx *gin.Context, conn *sqlx.DB) {
	var stop models.Stop

	if err := ctx.ShouldBindJSON(&stop); err != nil {
		ctx.JSON(400, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		log.Println("Ошибка парсинга:", err)
		return
	}

	_, err := conn.ExecContext(ctx, `
        INSERT INTO stops (name, lat, lng, radius, type, city, azimuth) 
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (id) DO UPDATE SET 
            name = EXCLUDED.name, 
            lat = EXCLUDED.lat, 
            lng = EXCLUDED.lng,
            azimuth = EXCLUDED.azimuth`,
		stop.Name, stop.Position[1], stop.Position[0], stop.Radius, stop.Type, stop.City, stop.Azimuth)

	if err != nil {
		log.Println("Не удалось добавить в базу данных новую остановку:", err)
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"status": "success"})
}

func EditBusStops(ctx *gin.Context, conn *sqlx.DB) {
	var stop models.Stop

	if err := ctx.ShouldBindJSON(&stop); err != nil {
		ctx.JSON(400, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		log.Println("Ошибка парсинга:", err)
		return
	}

	lat := stop.Position[0]
	lng := stop.Position[1]

	query := `
        UPDATE stops 
        SET name=$1, lat=$2, lng=$3, radius=$4, type=$5, azimuth=$6, city=$7 
        WHERE id=$8`

	_, err := conn.ExecContext(ctx, query,
		stop.Name,
		lat,
		lng,
		stop.Radius,
		stop.Type,
		stop.Azimuth,
		stop.City,
		stop.ID,
	)

	if err != nil {
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		log.Println("Ошибка парсинга:", err)
		return
	}

	ctx.JSON(200, gin.H{"status": "success"})
}

func FullBusStation(ctx *gin.Context, conn *sqlx.DB) {

	city := ctx.Param("city")
	var dbStops []models.Stop
	query := `SELECT id, name, lat, lng, radius, type, azimuth, city FROM stops WHERE city = $1`
	err := conn.SelectContext(ctx, &dbStops, query, city)
	if err != nil {
		log.Println("Ошибка парсинга:", err)
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
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
