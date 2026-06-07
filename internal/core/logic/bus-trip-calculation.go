package logic

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"orsavisionweb/internal/models"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
)

// Cчитает расстояние между автобусом и конечной в метрах
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

func ProcessTripState(db *sqlx.DB, busCtx *models.BusContext, currentPoint []float64, actualTime time.Time) {
	state := busCtx.State
	if state == nil {
		return
	}

	// 1. Ищем, находится ли автобус в радиусе какой-либо конечной прямо сейчас
	var activeFinalStop *models.Stop = nil
	for i := range busCtx.Stop {
		if busCtx.Stop[i].Type == "final" {
			dist := CalculateDistance(currentPoint[0], currentPoint[1], busCtx.Stop[i].Lat, busCtx.Stop[i].Lon)

			if dist <= float64(busCtx.Stop[i].Radius) {
				activeFinalStop = &busCtx.Stop[i]
				break
			}
		}
	}

	// 2. Логика, если мы НА конечной (Штатный заезд)
	if activeFinalStop != nil {
		if state.TripStatus == "in_trip" && state.CurrentStartStopID != activeFinalStop.ID {
			state.TripSequence++

			fromStopName := ""
			for _, s := range busCtx.Stop {
				if s.ID == state.CurrentStartStopID {
					fromStopName = s.Name
					break
				}
			}
			duration := actualTime.Sub(state.ActualDeparture)
			durationFactStr := fmt.Sprintf("%d мин", int(duration.Minutes()))

			delayMinutes := 0
			if state.PlanArrival != "" {
				parsedPlanStr, err := time.Parse("15:04:05", state.PlanArrival)
				if err == nil {
					planTime := time.Date(
						actualTime.Year(), actualTime.Month(), actualTime.Day(),
						parsedPlanStr.Hour(), parsedPlanStr.Minute(), parsedPlanStr.Second(), 0, actualTime.Location(),
					)
					if actualTime.After(planTime) {
						delayMinutes = int(actualTime.Sub(planTime).Minutes())
					}
				}
			}

			query := `
                INSERT INTO trip_reports (
                    city, report_date, route_number, bus_gov_number, trip_sequence,
                    from_stop_name, to_stop_name, planned_time, actual_time,
                    duration_fact, delay_minutes, status, created_at
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

			_, err := db.Exec(query,
				busCtx.City,
				actualTime.Format("2006-01-02"),
				busCtx.RouteNumber,
				busCtx.BusNumber,
				state.TripSequence,
				fromStopName,
				activeFinalStop.Name,
				state.PlanArrival,
				actualTime.Format("15:04:05"),
				durationFactStr,
				delayMinutes,
				"completed",
				actualTime,
			)
			if err != nil {
				log.Println("Не удалось записать оперативный отчет рейса:", err)
			}

			state.TripStatus = "idle"
			state.CurrentStartStopID = activeFinalStop.ID

		} else if state.TripStatus == "" || state.TripStatus == "idle" {
			state.TripStatus = "idle"
			state.CurrentStartStopID = activeFinalStop.ID
		}

		// 3. Логика, если мы В ПУТИ (вне конечных)
	} else {
		// ШТАТНЫЙ ВЫЕЗД: Если автобус стоял на конечной (idle) и вышел из её радиуса
		if state.TripStatus == "idle" {
			if state.CurrentStartStopID == 0 {
				return
			}

			state.TripStatus = "in_trip"
			state.ActualDeparture = actualTime
			log.Printf("Автобус %s (Маршрут %s) выехал с конечной ID %d в %s",
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
				log.Println("Не удалось достать расписание автобуса:", err)
			}

			state.PlanDeparture = planDep
			state.PlanArrival = planArr
		}

		// ====================================================================
		// ВРЕМЕННЫЙ ФОРСИРОВАННЫЙ СТАРТ ДЛЯ ТЕСТОВ
		// ====================================================================
		if state.TripStatus == "" {
			minDist := 99999999.0
			closestFinalStopID := 0

			// Ищем, какая конечная тупо ближе к нему географически прямо сейчас
			for i := range busCtx.Stop {
				if busCtx.Stop[i].Type == "final" {
					dist := CalculateDistance(currentPoint[0], currentPoint[1], busCtx.Stop[i].Lat, busCtx.Stop[i].Lon)
					if dist < minDist {
						minDist = dist
						closestFinalStopID = busCtx.Stop[i].ID
					}
				}
			}

			// Если нашли ближайшую конечную, принудительно открываем рейс прямо с дороги
			if closestFinalStopID != 0 {
				state.CurrentStartStopID = closestFinalStopID
				state.TripStatus = "in_trip"
				state.ActualDeparture = actualTime
				// Сразу подтягиваем под этот фейковый старт расписание, чтобы логика не путалась
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
					log.Println("Не удалось достать расписание при FORCE_START:", err)
				}
				state.PlanDeparture = planDep
				state.PlanArrival = planArr
			}
		}
		// ====================================================================
	}
}
