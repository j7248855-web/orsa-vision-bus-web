package utils

import (
	"orsavisionweb/internal/database"
	"orsavisionweb/internal/models"
	"time"
)

func LoadFullBusData(ip string) *models.BusContext {
	conn := database.Connection()
	ctx := &models.BusContext{}

	conn.Get(ctx, `
		SELECT b.id as bus_id, b.route_number as route_number 
		FROM devices d
		JOIN buses b ON d.bus_id = b.id
		WHERE d.rtsp_link = $1 AND d.type = 'teltonic' 
		LIMIT 1`, ip)

	conn.Select(&ctx.Stop, `
		SELECT id, name, lat, lng, radius, type, city 
		FROM stops 
		WHERE route_id = (SELECT id FROM routes WHERE route_number = $1 LIMIT 1)
		ORDER BY sequence_order ASC`, ctx.RouteNumber)
	dayType := "workday"
	now := time.Now()
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		dayType = "weekend"
	}
	type tempSchedule struct {
		StopID int    `db:"stop_id"`
		Time   string `db:"arrival_time"`
	}
	var scheduleRows []tempSchedule

	//Вытягиваем всё расписание для этого автобуса на текущий тип дня
	conn.Select(&scheduleRows, `
        SELECT stop_id, arrival_time 
        FROM stop_schedules 
        WHERE bus_id = $1 AND day_type = $2`, ctx.BusID, dayType)

	// Распределяем времена по остановкам в памяти
	for i := range ctx.Stop {
		for _, sch := range scheduleRows {
			if sch.StopID == ctx.Stop[i].ID {
				ctx.Stop[i].Schedule = append(ctx.Stop[i].Schedule, sch.Time)
			}
		}
	}
	var points []struct {
		Lat float64 `db:"lat"`
		Lng float64 `db:"lng"`
	}

	err := conn.Select(&points, `
		SELECT lat, lng 
		FROM route_path_points 
		WHERE route_id = (SELECT id FROM routes WHERE route_number = $1 LIMIT 1)
		ORDER BY sequence_order ASC`, ctx.RouteNumber)

	if err == nil {
		ctx.Points = make([][2]float64, len(points))
		for i, p := range points {
			ctx.Points[i] = [2]float64{p.Lng, p.Lat}
		}
	}

	ctx.State = &models.Dependence{}

	return ctx
}
