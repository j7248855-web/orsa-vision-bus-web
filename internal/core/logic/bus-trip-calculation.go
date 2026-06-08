package logic

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"orsavisionweb/internal/models"
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

			if dist <= float64(busCtx.Stop[i].Radius) {
				activeFinalStop = &busCtx.Stop[i]
				log.Printf("[DEBUG] Автобус %s внутри конечной: %s", busCtx.BusNumber, activeFinalStop.Name)
				break
			}
		}
	}

	if activeFinalStop != nil {
		log.Printf("[DEBUG] Проверка финиша. Статус: %s, Стартовая остановка: %d, Текущая: %d",
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
			diffMinutes := int(duration.Minutes())
			if diffMinutes < 0 {
				diffMinutes = 0
			}
			durationFactStr := fmt.Sprintf("%d мин", diffMinutes)

			delayMinutes := 0
			if state.PlanArrival != "" {
				parsedPlanStr, err := time.Parse("15:04:05", state.PlanArrival)
				if err == nil {
					planTime := time.Date(
						actualTime.Year(), actualTime.Month(), actualTime.Day(),
						parsedPlanStr.Hour(), parsedPlanStr.Minute(), parsedPlanStr.Second(), 0,
						actualTime.Location(),
					)

					if actualTime.After(planTime) {
						delayMinutes = int(actualTime.Sub(planTime).Minutes())
					}
				}
			}

			var safePlannedTime interface{} = nil
			if state.PlanArrival != "" {
				safePlannedTime = state.PlanArrival
			}

			query := `
				INSERT INTO trip_reports (
					city, report_date, route_number, bus_gov_number, trip_sequence,
					from_stop_name, to_stop_name, planned_time, actual_time,
					duration_fact, delay_minutes, status, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

			log.Printf("[DEBUG] Запись отчета. Автобус: %s, График: %d, План финиша: %v, Факт: %s, Задержка: %d",
				busCtx.BusNumber, busCtx.SequenceNumber, safePlannedTime, actualTime.Format("15:04:05"), delayMinutes)

			_, err := db.Exec(query,
				busCtx.City,
				actualTime.Format("2006-01-02"),
				busCtx.RouteNumber,
				busCtx.BusNumber,
				state.TripSequence,
				fromStopName,
				activeFinalStop.Name,
				safePlannedTime,
				actualTime.Format("15:04:05"),
				durationFactStr,
				delayMinutes,
				"completed",
				actualTime,
			)
			if err != nil {
				log.Println("[ERROR] Ошибка INSERT в trip_reports:", err)
			} else {
				log.Println("[SUCCESS] Отчет рейса записан!")
			}

			state.TripStatus = "idle"
			state.CurrentStartStopID = activeFinalStop.ID

		} else if state.TripStatus == "" || state.TripStatus == "idle" {
			state.TripStatus = "idle"
			state.CurrentStartStopID = activeFinalStop.ID
		}

	} else {
		// Автобус находится на маршруте (между конечными)
		if state.TripStatus == "idle" {
			if state.CurrentStartStopID == 0 {
				return
			}

			state.TripStatus = "in_trip"
			state.ActualDeparture = actualTime
			log.Printf("[DEBUG] ТРИГГЕР СТАРТА: Автобус %s (График %d) выехал с конечной ID %d в %s",
				busCtx.BusNumber, busCtx.SequenceNumber, state.CurrentStartStopID, actualTime.Format("15:04:05"))

			var planDepStr, planArrStr string

			// Логика обычного старта: ищем рейс по его графику, ближайший по времени ОТПРАВЛЕНИЯ
			query := `
				SELECT s.departure_time::text, s.arrival_time::text 
				FROM schedules s
				JOIN routes r ON s.route_id = r.id
				WHERE r.route_number = $1 AND s.sequence_number = $2
				ORDER BY ABS(EXTRACT(EPOCH FROM (s.departure_time::time - $3::time))) 
				LIMIT 1`

			err := db.QueryRow(query, busCtx.RouteNumber, busCtx.SequenceNumber, actualTime.Format("15:04:05")).Scan(&planDepStr, &planArrStr)
			if err != nil {
				if err != sql.ErrNoRows {
					log.Println("[ERROR] Не удалось достать расписание при старте:", err)
				}
				state.PlanDeparture = ""
				state.PlanArrival = ""
			} else {
				if len(planDepStr) > 8 {
					planDepStr = planDepStr[:8]
				}
				if len(planArrStr) > 8 {
					planArrStr = planArrStr[:8]
				}

				state.PlanDeparture = planDepStr
				state.PlanArrival = planArrStr
				log.Printf("[DEBUG] Найдено расписание: План старт=%s, План финиш=%s", state.PlanDeparture, state.PlanArrival)
			}
		}

		// РЕТРОСПЕКТИВНАЯ ИНИЦИАЛИЗАЦИЯ (Старт из "никуда")
		if state.TripStatus == "" {
			log.Printf("[DEBUG] Инициализация автобуса %s (График %d) на маршруте...", busCtx.BusNumber, busCtx.SequenceNumber)
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

				var planDepStr, planArrStr string

				// УМНЫЙ ЗАПРОС: Ищем рейс строго по графику автобуса,
				// ориентируясь на то, к какому ПРИБЫТИЮ (arrival_time) ближе всего текущее время на часах
				query := `
					SELECT s.departure_time::text, s.arrival_time::text 
					FROM schedules s
					JOIN routes r ON s.route_id = r.id
					WHERE r.route_number = $1 AND s.sequence_number = $2
					ORDER BY ABS(EXTRACT(EPOCH FROM (s.arrival_time::time - $3::time))) 
					LIMIT 1`

				err := db.QueryRow(query, busCtx.RouteNumber, busCtx.SequenceNumber, actualTime.Format("15:04:05")).Scan(&planDepStr, &planArrStr)
				if err != nil {
					if err != sql.ErrNoRows {
						log.Println("[ERROR] Не удалось достать расписание ретроспективно:", err)
					}
					state.PlanDeparture = ""
					state.PlanArrival = ""
				} else {
					if len(planDepStr) > 8 {
						planDepStr = planDepStr[:8]
					}
					if len(planArrStr) > 8 {
						planArrStr = planArrStr[:8]
					}

					state.PlanDeparture = planDepStr
					state.PlanArrival = planArrStr
					log.Printf("[DEBUG] Ретроспективно закреплен план: %s -> %s", state.PlanDeparture, state.PlanArrival)
				}
			}
		}
	}
}
