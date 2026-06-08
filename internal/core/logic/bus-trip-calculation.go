package logic

import (
	"fmt"
	"log"
	"math"
	"orsavisionweb/internal/models"
	"time"

	"github.com/jmoiron/sqlx"
)

// Получает локацию времени на основе города Казахстана
func GetLocationByCity(city string) *time.Location {
	var zoneName string
	switch city {
	case "Актобе", "Атырау", "Актау", "Уральск", "Кызылорда":
		zoneName = "Asia/Aqtobe"
	case "Астана", "Алматы", "Шымкент", "Караганда", "Павлодар", "Усть-Каменогорск", "Тараз", "Костанай", "Петропавловск", "Семей":
		zoneName = "Asia/Almaty"
	default:
		zoneName = "Asia/Almaty"
	}

	loc, err := time.LoadLocation(zoneName)
	if err != nil {
		return time.FixedZone("KZTS", 5*60*60)
	}
	return loc
}

// Cчитает расстояние между автобусом и конечной в метрах
func CalculateDistance(busLat, busLon, finalStopLat, finalStopLon float64) float64 {
	const R = 6371000
	const rad = math.Pi / 180
	lat1 := busLat * rad
	lon1 := busLon * rad
	lat2 := finalStopLat * rad // Исправлено: берем lat конечной, а не автобуса повторно
	lon2 := finalStopLon * rad
	dLat := lat2 - lat1
	dLon := lon2 - lon1
	calculate := 2 * R * math.Asin(math.Sqrt(
		math.Pow(math.Sin(dLat/2), 2)+
			math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dLon/2), 2),
	))
	return calculate
}

// Вспомогательная функция для генерации зигзаг-расписания на лету
func SliceScheduleZigZag(db *sqlx.DB, routeNumber string, sequenceNumber int, actualTime time.Time) (string, string) {
	timeStr := actualTime.Format("15:04:05")

	query := `
		SELECT s.departure_time::text, s.arrival_time::text 
		FROM schedules s
		JOIN routes r ON s.route_id = r.id
		WHERE r.route_number = $1 AND s.sequence_number = $2
		ORDER BY s.departure_time::time ASC`

	rows, err := db.Query(query, routeNumber, sequenceNumber)
	if err != nil {
		log.Println("[ERROR] Ошибка Query в SliceScheduleZigZag:", err)
		return "", ""
	}
	defer rows.Close()

	type scheduleRow struct {
		dep string
		arr string
	}

	var list []scheduleRow
	for rows.Next() {
		var r scheduleRow
		if err := rows.Scan(&r.dep, &r.arr); err == nil {
			if len(r.dep) > 8 {
				r.dep = r.dep[:8]
			}
			if len(r.arr) > 8 {
				r.arr = r.arr[:8]
			}
			list = append(list, r)
		}
	}

	if len(list) == 0 {
		return "", ""
	}

	for i, currentTrip := range list {
		if i == len(list)-1 {
			return currentTrip.dep, currentTrip.arr
		}

		nextTrip := list[i+1]

		// Зигзаг между рейсами
		if timeStr >= currentTrip.arr && timeStr <= nextTrip.dep {
			log.Printf("[DEBUG] ЗИГЗАГ ДЕТЕКТЕД: Время %s между рейсами %s и %s.", timeStr, currentTrip.arr, nextTrip.dep)
			return currentTrip.arr, nextTrip.dep
		}

		// Стандартный рейс по расписанию
		if timeStr >= currentTrip.dep && timeStr < currentTrip.arr {
			return currentTrip.dep, currentTrip.arr
		}
	}

	return list[0].dep, list[0].arr
}

func ProcessTripState(db *sqlx.DB, busCtx *models.BusContext, currentPoint []float64, actualTime time.Time) {
	state := busCtx.State
	if state == nil {
		log.Println("[DEBUG] State у автобуса nil")
		return
	}

	loc := GetLocationByCity(busCtx.City)
	actualTime = actualTime.In(loc)

	// Ищем активную конечную остановку, в радиусе которой находится автобус
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

	if activeFinalStop != nil {
		// АВТОБУС НА КОНЕЧНОЙ
		if state.TripStatus == "in_trip" {
			// Проверяем, действительно ли он сменил конечную, чтобы исключить ложные триггеры
			if state.CurrentStartStopID == activeFinalStop.ID {
				log.Printf("[REJECT] Отчет НЕ записан. Автобус на той же конечной, откуда стартовал (ID: %d). Ждем смены направления.", activeFinalStop.ID)
				return
			}

			state.TripSequence++

			fromStopName := "Неизвестная остановка"
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

			log.Printf("[DEBUG] Попытка INSERT оперативного отчета. План финиша: %v, Факт: %s", safePlannedTime, actualTime.Format("15:04:05"))

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
				log.Println("[DATABASE ERROR] Ошибка записи в trip_reports:", err)
			} else {
				log.Printf("[SUCCESS] ОПЕРАТИВНЫЙ ОТЧЕТ УСПЕШНО ЗАПИСАН! Автобус: %s, Из: %s -> В: %s, План: %v, Факт: %s",
					busCtx.BusNumber, fromStopName, activeFinalStop.Name, safePlannedTime, actualTime.Format("15:04:05"))
			}

			state.TripStatus = "idle"
			state.CurrentStartStopID = activeFinalStop.ID

		} else {
			// Если статус idle или пустой — автобус просто стоит на конечной
			state.TripStatus = "idle"
			state.CurrentStartStopID = activeFinalStop.ID
		}

	} else {
		// АВТОБУС В ПУТИ (МЕЖДУ КОНЕЧНЫМИ)
		if state.TripStatus == "idle" {
			if state.CurrentStartStopID == 0 {
				return
			}

			state.TripStatus = "in_trip"
			state.ActualDeparture = actualTime
			log.Printf("[DEBUG] ТРИГГЕР СТАРТА: Автобус %s выехал с конечной ID %d в %s",
				busCtx.BusNumber, state.CurrentStartStopID, actualTime.Format("15:04:05"))

			dep, arr := SliceScheduleZigZag(db, busCtx.RouteNumber, busCtx.SequenceNumber, actualTime)
			state.PlanDeparture = dep
			state.PlanArrival = arr
			log.Printf("[DEBUG] Закреплено плановое время точки прибытия по зигзагу: %s -> %s", state.PlanDeparture, state.PlanArrival)
		}

		// ИСПРАВЛЕННАЯ РЕТРОСПЕКТИВНАЯ ИНИЦИАЛИЗАЦИЯ (Старт из "никуда")
		if state.TripStatus == "" {
			log.Printf("[DEBUG] Инициализация из никуда. Автобус %s, время %s...", busCtx.BusNumber, actualTime.Format("15:04:05"))

			dep, arr := SliceScheduleZigZag(db, busCtx.RouteNumber, busCtx.SequenceNumber, actualTime)
			state.PlanDeparture = dep
			state.PlanArrival = arr

			// Ищем БЛИЖАЙШУЮ конечную — это точка, КУДА он едет (цель финиша)
			minDist := 99999999.0
			targetFinalStopID := 0
			for i := range busCtx.Stop {
				if busCtx.Stop[i].Type == "final" {
					dist := CalculateDistance(currentPoint[0], currentPoint[1], busCtx.Stop[i].Lat, busCtx.Stop[i].Lon)
					if dist < minDist {
						minDist = dist
						targetFinalStopID = busCtx.Stop[i].ID
					}
				}
			}

			// Находим противоположную конечную (откуда он выехал)
			sourceStopID := 0
			for _, s := range busCtx.Stop {
				if s.Type == "final" && s.ID != targetFinalStopID {
					sourceStopID = s.ID
					break
				}
			}

			// ЕслиsourceStopID не нашли (в массиве всего 1 конечная), берем target
			if sourceStopID == 0 {
				sourceStopID = targetFinalStopID
			}

			state.CurrentStartStopID = sourceStopID
			state.TripStatus = "in_trip"
			state.ActualDeparture = actualTime

			log.Printf("[DEBUG] Ретроспективно определено направление: Из ID %d -> В ID %d. План зигзага: %s -> %s",
				state.CurrentStartStopID, targetFinalStopID, state.PlanDeparture, state.PlanArrival)
		}
	}
}
