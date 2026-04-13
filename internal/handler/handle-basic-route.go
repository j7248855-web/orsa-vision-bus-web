package handler

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func HandleRouteWithPoints(ctx *gin.Context, conn *sqlx.DB) {
	var route models.Route

	if err := ctx.ShouldBindJSON(&route); err != nil {
		log.Printf("[ERROR] Parse JSON: %v", err)
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	tx, err := conn.Beginx()
	if err != nil {
		log.Printf("[ERROR] Begin Transaction: %v", err)
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	var actualRouteID int
	err = tx.QueryRow(`
        INSERT INTO routes (route_number, name, city, status) 
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (route_number, city) DO UPDATE SET 
            name = EXCLUDED.name, 
            status = EXCLUDED.status
        RETURNING id`,
		route.RouteID, route.Name, route.City, route.Status).Scan(&actualRouteID)

	if err != nil {
		log.Printf("[ERROR] Insert/Update Route (Number: %s): %v", route.RouteID, err)
		ctx.JSON(500, gin.H{"error": "DB Route Error: " + err.Error()})
		return
	}

	_, err = tx.Exec("DELETE FROM route_path_points WHERE route_id = $1", actualRouteID)
	if err != nil {
		log.Printf("[ERROR] Delete Old Points (ID: %d): %v", actualRouteID, err)
		ctx.JSON(500, gin.H{"error": "Clear Points Error: " + err.Error()})
		return
	}

	for i, p := range route.Points {
		// ВНИМАНИЕ: Проверь название таблицы ниже (было report_path_points, может route_path_points?)
		_, err = tx.Exec(`
            INSERT INTO route_path_points (route_id, lat, lng, sequence_order) 
            VALUES ($1, $2, $3, $4)`,
			actualRouteID, p[1], p[0], i)

		if err != nil {
			log.Printf("[ERROR] Insert Point %d for Route %d: %v", i, actualRouteID, err)
			ctx.JSON(500, gin.H{"error": "Insert Point Error: " + err.Error()})
			return
		}
	}

	_, err = tx.Exec("DELETE FROM route_stops WHERE route_id = $1", actualRouteID)
	if err != nil {
		log.Printf("[ERROR] Delete Old Stops (ID: %d): %v", actualRouteID, err)
		ctx.JSON(500, gin.H{"error": "Clear Stops Error: " + err.Error()})
		return
	}

	for i, stop := range route.Stops {
		_, err = tx.Exec(`
            INSERT INTO route_stops (route_id, stop_id, sequence_order) 
            VALUES ($1, $2, $3)`,
			actualRouteID, stop.ID, i)

		if err != nil {
			log.Printf("[ERROR] Link Stop ID %d to Route %d: %v", stop.ID, actualRouteID, err)
			ctx.JSON(500, gin.H{"error": "Link Stop Error: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[ERROR] Commit Transaction: %v", err)
		ctx.JSON(500, gin.H{"error": "Commit Error: " + err.Error()})
		return
	}

	log.Printf("[SUCCESS] Route %s saved. ID: %d, Points: %d, Stops: %d", route.RouteID, actualRouteID, len(route.Points), len(route.Stops))
	ctx.JSON(200, gin.H{"status": "success", "id": actualRouteID})
}
