package logic

import (
	"fmt"
	"log"
	"math"
	"orsavisionweb/internal/models"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	MetersPerDegree = 111111.0
	MaxAllowedDev   = 1000.0
	MaxGPSThresh    = 5000.0
)

// Высчитывает минимальное расстояние до геометрии маршрута
func GetMinDeviationMeters(lat, lon float64, routePoints [][2]float64) float64 {
	if len(routePoints) < 2 {
		return 0
	}
	var minDeviation = math.MaxFloat64

	busX := lon
	busY := lat

	for i := 0; i < len(routePoints)-1; i++ {
		p1 := routePoints[i]
		p2 := routePoints[i+1]

		p1X, p1Y := p1[1], p1[0]
		p2X, p2Y := p2[1], p2[0]

		vecRoadX := p2X - p1X
		vecRoadY := p2Y - p1Y

		vecToBusX := busX - p1X
		vecToBusY := busY - p1Y

		dotProduct := (vecToBusX * vecRoadX) + (vecToBusY * vecRoadY)
		magRoadSq := (vecRoadX * vecRoadX) + (vecRoadY * vecRoadY)

		if magRoadSq == 0 {
			continue
		}

		t := dotProduct / magRoadSq
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}

		projX := p1X + (vecRoadX * t)
		projY := p1Y + (vecRoadY * t)

		deltaLat := busY - projY
		deltaLon := busX - projX

		metersY := deltaLat * MetersPerDegree
		busRad := busY * math.Pi / 180.0
		metersX := deltaLon * MetersPerDegree * math.Cos(busRad)

		currentDev := math.Sqrt(math.Pow(metersX, 2) + math.Pow(metersY, 2))
		if currentDev < minDeviation {
			minDeviation = currentDev
		}
	}

	if minDeviation == math.MaxFloat64 {
		return 0
	}
	return minDeviation
}

// Управляет состояниями схода/возврата и избавляет базу от спама
func ProcessRouteDeviation(db *sqlx.DB, busCtx *models.BusContext, currentPoint []float64, actualTime time.Time) {
	state := busCtx.State
	if state == nil {
		log.Println("[WARN] State для автобуса nil, пропускаем расчет отклонения")
		return
	}

	currentDev := GetMinDeviationMeters(currentPoint[0], currentPoint[1], busCtx.Points)

	if currentDev > MaxGPSThresh {
		log.Printf("[GPS FLOOD] Игнорируем аномальный прыжок GPS для автобуса %s. Смещение: %.2f м", busCtx.BusNumber, currentDev)
		return
	}

	if actualTime.Year() < 2020 {
		actualTime = time.Now()
	}

	if currentDev > MaxAllowedDev {
		if !state.IsOffRoute {
			state.IsOffRoute = true
			state.MaxDeviation = currentDev
			state.OffRouteStartTime = actualTime

			log.Printf("[ALERT] 🚨 Автобус %s (Маршрут %s) СОШЕЛ С МАРШРУТА! Смещение: %.2f м",
				busCtx.BusNumber, busCtx.RouteNumber, currentDev)
		} else {
			if currentDev > state.MaxDeviation {
				state.MaxDeviation = currentDev
			}
		}
	} else {
		if state.IsOffRoute {
			if state.OffRouteStartTime.Year() < 2020 {
				state.OffRouteStartTime = actualTime.Add(-5 * time.Second)
			}

			duration := actualTime.Sub(state.OffRouteStartTime)
			durationMinutes := int(duration.Minutes())

			if durationMinutes < 0 || durationMinutes > 1440 {
				durationMinutes = 1
			}

			log.Printf("[SUCCESS] ✅ Автобус %s ВЕРНУЛСЯ на маршрут. Гулял: %d мин. Пиковое смещение: %.2f м",
				busCtx.BusNumber, durationMinutes, state.MaxDeviation)

			description := fmt.Sprintf("Гулял: %d мин. Пиковое смещение: %.2f м", durationMinutes, state.MaxDeviation)

			query := `
				INSERT INTO bus_violations (
					bus_id, route_num, plate_num, violation_type, value, created_at
				) VALUES ($1, $2, $3, $4, $5, $6)`

			_, err := db.Exec(query,
				busCtx.BusID,
				busCtx.RouteNumber,
				busCtx.BusNumber,
				"Сход с маршрута",
				description,
				actualTime,
			)
			if err != nil {
				log.Println("[DATABASE ERROR] Ошибка записи инцидента отклонения:", err)
			} else {
				log.Printf("[SUCCESS] Инцидент для автобуса %s успешно залогирован в БД.", busCtx.BusNumber)
			}

			state.IsOffRoute = false
			state.MaxDeviation = 0
		}
	}
}
