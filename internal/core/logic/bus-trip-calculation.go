package logic

import (
	"database/sql"
	"log"
	"math"
	"orsavisionweb/internal/models"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
)

// CalculateDistance считает расстояние между автобусом и конечной в метрах
func CalculateDistance(busLat, busLon, finalStopLat, finalStopLon float64) float64 {
	const R = 6371000
	const rad = math.Pi / 180
	lat1 := busLat * rad
	lon1 := busLon * rad
	lat2 := finalStopLat * rad
	lon2 := finalStopLon * rad
	dLat := lat2 - lat1
	dLon := lon2 - lon1
	calculate := 2 * R * math.Asin(math.Sqrt(
		math.Pow(math.Sin(dLat/2), 2)+
			math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dLon/2), 2),
	))
	return calculate
}

// ProcessTripState — это стейт-машина контроля рейсов автобуса
func ProcessTripState(db *sqlx.DB, busCtx *models.BusContext, currentPoint []float64, actualTime time.Time) {
	state := busCtx.State
	if state == nil {
		return
	}

	var activeFinalStop *models.Stop = nil
	for i := range busCtx.Stop {
		// Проверяем только остановки с типом "final"
		if busCtx.Stop[i].Type == "final" {
			dist := CalculateDistance(currentPoint[0], currentPoint[1], busCtx.Stop[i].Lat, busCtx.Stop[i].Lon)

			// Если расстояние меньше или равно радиусу из базы
			if dist <= float64(busCtx.Stop[i].Radius) {
				activeFinalStop = &busCtx.Stop[i]
				break // Конечная найдена
			}
		}
	}
	if activeFinalStop != nil {

		if state.TripStatus == "in_trip" && state.CurrentStartStopID != activeFinalStop.ID {
			log.Printf("[РЕЙС ФИНИШ] Автобус %s (Маршрут %s) прибыл на конечную %s", busCtx.BusNumber, busCtx.RouteNumber, activeFinalStop.Name)

			query := `
				INSERT INTO trip_reports (
					bus_id, route_number, from_stop_id, to_stop_id, 
					plan_start_time, actual_start_time, plan_end_time, actual_end_time
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

			_, err := db.Exec(query,
				busCtx.BusID,
				busCtx.RouteNumber,
				state.CurrentStartStopID, // ID конечной старта (запомнили ранее)
				activeFinalStop.ID,       // ID конечной финиша (где стоим сейчас)
				state.PlanDeparture,      // Плановое время выезда из памяти
				state.ActualDeparture,    // Реальное время выезда из памяти
				state.PlanArrival,        // Плановое время приезда из памяти
				actualTime,               // Реальное время приезда (сейчас по GPS)
			)
			if err != nil {
				log.Println("[ОШИБКА БД] Не удалось записать отчет рейса:", err)
			}

			// После записи отчета переводим автобус в режим отстоя
			state.TripStatus = "idle"
			state.CurrentStartStopID = activeFinalStop.ID

		} else if state.TripStatus == "" {
			state.TripStatus = "idle"
			state.CurrentStartStopID = activeFinalStop.ID
		}

	} else {

		if state.TripStatus == "idle" || state.TripStatus == "" {

			if state.CurrentStartStopID == 0 {
				return
			}
			state.TripStatus = "in_trip"
			state.ActualDeparture = actualTime
			log.Printf("[РЕЙС СТАРТ] Автобус %s (Маршрут %s) выехал с конечной ID %d в %s",
				busCtx.BusNumber, busCtx.RouteNumber, state.CurrentStartStopID, actualTime.Format("15:04:05"))
			routeIDInt, _ := strconv.Atoi(busCtx.RouteNumber)

			var planDep, planArr string

			query := `
				SELECT departure_time, arrival_time 
				FROM schedules 
				WHERE route_id = $1 AND sequence_number = $2
				ORDER BY ABS(EXTRACT(EPOCH FROM (departure_time - $3::time))) 
				LIMIT 1`

			err := db.QueryRow(query, routeIDInt, busCtx.SequenceNumber, actualTime.Format("15:04:05")).Scan(&planDep, &planArr)
			if err != nil && err != sql.ErrNoRows {
				log.Println("[ОШИБКА БД] Не удалось достать расписание автобуса:", err)
			}

			state.PlanDeparture = planDep
			state.PlanArrival = planArr
		}
	}
}
