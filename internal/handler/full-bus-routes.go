package handler

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func GetFullRoutes(ctx *gin.Context, conn *sqlx.DB) {
	var routes []models.Route

	err := conn.Select(&routes, "SELECT id, route_number, name, status FROM routes")
	if err != nil {
		log.Println("Ошибка получения маршрутов:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка базы данных"})
		return
	}

	for i := range routes {
		routeID := routes[i].ID

		var points []struct {
			Lat float64 `db:"lat"`
			Lng float64 `db:"lng"`
		}
		err = conn.Select(&points, `
            SELECT lat, lng FROM route_path_points 
            WHERE route_id = $1 ORDER BY sequence_order`, routeID)

		if err == nil {
			routes[i].Points = make([][2]float64, 0, len(points))
			for _, p := range points {
				routes[i].Points = append(routes[i].Points, [2]float64{p.Lng, p.Lat})
			}
		}

		var stops []models.Stop
		err = conn.Select(&stops, `
            SELECT s.id, s.name, s.lat, s.lng, s.radius, s.type, s.azimuth, s.city 
            FROM stops s
            JOIN route_stops rs ON s.id = rs.stop_id
            WHERE rs.route_id = $1
            ORDER BY rs.sequence_order`, routeID)

		if err == nil {
			for j := range stops {
				stops[j].Position = [2]float64{stops[j].Lon, stops[j].Lat}
			}
			routes[i].Stops = stops
		} else {
			log.Printf("Ошибка получения остановок для маршрута %s: %v", routeID, err)
			routes[i].Stops = []models.Stop{} // Чтобы не было nil в JSON
		}
	}

	if routes == nil {
		routes = []models.Route{}
	}

	ctx.JSON(200, routes)
}
