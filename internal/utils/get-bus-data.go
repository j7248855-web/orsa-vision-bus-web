package utils

import (
	"log"
	"orsavisionweb/internal/database"
	"orsavisionweb/internal/models"
)

func LoadFullBusData(ip string) *models.BusContext {
	conn := database.Connection()
	ctx := &models.BusContext{}

	err := conn.Get(ctx, `
		SELECT b.id as bus_id, b.route_number as route_number 
		FROM devices d
		JOIN buses b ON d.bus_id = b.id
		WHERE d.rtsp_link = $1 AND d.type = 'teltonic' 
		LIMIT 1`, ip)

	if err != nil {
		log.Printf("Не были найдены девайсы по этому IP %s: %v", ip, err)
		return nil
	}

	err = conn.Select(&ctx.Stop, `
		SELECT id, name, lat, lng, radius, type, city 
		FROM stops 
		WHERE route_id = (SELECT id FROM routes WHERE route_number = $1 LIMIT 1)
		ORDER BY sequence_order ASC`, ctx.RouteNumber)

	if err != nil {
		log.Printf("Остановки не были найдены по этому маршруту %s: %v", ctx.RouteNumber, err)
	}

	var points []struct {
		Lat float64 `db:"lat"`
		Lng float64 `db:"lng"`
	}

	err = conn.Select(&points, `
		SELECT lat, lng 
		FROM route_path_points 
		WHERE route_id = (SELECT id FROM routes WHERE route_number = $1 LIMIT 1)
		ORDER BY sequence_order ASC`, ctx.RouteNumber)

	if err == nil {
		ctx.Points = make([][2]float64, len(points))
		for i, p := range points {
			ctx.Points[i] = [2]float64{p.Lat, p.Lng}
		}
	}

	// 4. Инициализируем состояние для расчетов
	ctx.State = &models.Dependence{}

	return ctx
}
