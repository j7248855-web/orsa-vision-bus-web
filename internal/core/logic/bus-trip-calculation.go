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
		log.Println("[DEBUG] State у автобуса nil")
		return
	}

	var activeFinalStop *models.Stop = nil
	for i := range busCtx.Stop {
		if busCtx.Stop[i].Type == "final" {
			dist := CalculateDistance(currentPoint[0], currentPoint[1], busCtx.Stop[i].Lat, busCtx.Stop[i].Lon)

			// Лог для проверки геометрии
			log.Printf("[DEBUG] Автобус %s до конечной %s (%d) = %.2f м. Радиус: %d",
				busCtx.BusNumber, busCtx.Stop[i].Name, busCtx.Stop[i].ID, dist, busCtx.Stop[i].Radius)

			if dist <= float64(busCtx.Stop[i].Radius) {
				activeFinalStop = &busCtx.Stop[i]
				log.Printf("[DEBUG] Автобус внутри конечной: %s", activeFinalStop.Name)
				break
			}
		}
	}

	if activeFinalStop != nil {
		log.Printf("[DEBUG] Проверка триггера финиша. Текущий статус: %s, Стартовый ID конечной: %d, Текущий ID конечной: %d",
			state.TripStatus, state.CurrentStartStopID, activeFinalStop.ID)

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

			log.Printf("[DEBUG] Пробуем выполнить INSERT для рейса %d...", state.TripSequence)
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
				log.Println("[ERROR] Ошибка выполнения INSERT в trip_reports:", err)
			} else {
				log.Println("[SUCCESS] Отчет рейса успешно записан в базу!")
			}

			state.TripStatus = "idle"
			state.CurrentStartStopID = activeFinalStop.ID

		} else if state.TripStatus == "" || state.TripStatus == "idle" {
			state.TripStatus = "idle"
			state.CurrentStartStopID = activeFinalStop.ID
		}

	} else {
		// Логика когда автобус вне конечных остановок
		if state.TripStatus == "idle" {
			if state.CurrentStartStopID == 0 {
				return
			}

			state.TripStatus = "in_trip"
			state.ActualDeparture = actualTime
			log.Printf("[DEBUG] ТРИГГЕР СТАРТА: Автобус %s выехал с конечной ID %d в %s",
				busCtx.BusNumber, state.CurrentStartStopID, actualTime.Format("15:04:05"))

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
				log.Println("[ERROR] Не удалось достать расписание:", err)
			}

			state.PlanDeparture = planDep
			state.PlanArrival = planArr
		}

		if state.TripStatus == "" {
			log.Println("[DEBUG] Статус пустой, инициализируем принудительный старт с маршрута...")
			minDist := 99999999.0
			closestFinalStopID := 0

			for i := range busCtx.Stop {
				if busCtx.Stop[i].Type == "final" {
					dist := CalculateDistance(currentPoint[0], currentPoint[1], busCtx.Stop[i].Lat, busCtx.Stop[i].Lon)
					if dist < minDist {
						minDist = dist
						closestFinalStopID = busCtx.Stop[i].ID
					}
				}
			}

			if closestFinalStopID != 0 {
				state.CurrentStartStopID = closestFinalStopID
				state.TripStatus = "in_trip"
				state.ActualDeparture = actualTime

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
					log.Println("[ERROR] Не удалось достать расписание при FORCE_START:", err)
				}
				state.PlanDeparture = planDep
				state.PlanArrival = planArr
			}
		}
	}
}
