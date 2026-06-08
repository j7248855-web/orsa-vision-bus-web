package logic

import (
	"log"
	"math"
	"orsavisionweb/internal/models"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	MetersPerDegree = 111111.0
	MaxAllowedDev   = 200.0 // Твой лимит отклонения
)

// Высчитывает минимальное расстояние до геометрии маршрута (твоя математика)
func GetMinDeviationMeters(lat, lon float64, routePoints [][2]float64) float64 {
	if len(routePoints) < 2 {
		return 0
	}
	var minDeviation = math.MaxFloat64
	busPos := []float64{lat, lon}

	for i := 0; i < len(routePoints)-1; i++ {
		p1 := routePoints[i]
		p2 := routePoints[i+1]

		// Вектор сегмента дороги
		vecRoad := []float64{p2[0] - p1[0], p2[1] - p1[1]}
		// Вектор от старта сегмента до автобуса
		vecToBus := []float64{busPos[0] - p1[0], busPos[1] - p1[1]}

		dotProduct := (vecToBus[0] * vecRoad[0]) + (vecToBus[1] * vecRoad[1])
		magRoadSq := (vecRoad[0] * vecRoad[0]) + (vecRoad[1] * vecRoad[1])

		if magRoadSq == 0 {
			continue
		}

		t := dotProduct / magRoadSq

		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}

		// Точка проекции на отрезок
		projection := []float64{
			p1[0] + (vecRoad[0] * t),
			p1[1] + (vecRoad[1] * t),
		}

		// Дельты координат (исправлен порядок вычитания для осей)
		deltaLat := busPos[0] - projection[0]
		deltaLon := busPos[1] - projection[1]

		metersY := deltaLat * MetersPerDegree
		busRad := busPos[0] * math.Pi / 180.0 // Радианы берем от широты (Lat - первый элемент)
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

	// Считаем метры по твоим точкам маршрута (busCtx.Points)
	currentDev := GetMinDeviationMeters(currentPoint[0], currentPoint[1], busCtx.Points)

	if currentDev > MaxAllowedDev {
		// --- АВТОБУС ВНЕ МАРШРУТА ---
		if !state.IsOffRoute {
			// ТРИГГЕР: Момент первого схода с маршрута
			state.IsOffRoute = true
			state.MaxDeviation = currentDev
			state.OffRouteStartTime = actualTime

			log.Printf("[ALERT] 🚨 Автобус %s (Маршрут %s) СОШЕЛ С МАРШРУТА! Смещение: %.2f м",
				busCtx.BusNumber, busCtx.RouteNumber, currentDev)
		} else {
			// Автобус УЖЕ едет вне маршрута.
			// Просто обновляем пиковое значение в оперативной памяти (ОЗУ). В БАЗУ НИЧЕГО НЕ ПИШЕМ!
			if currentDev > state.MaxDeviation {
				state.MaxDeviation = currentDev
			}
		}
	} else {
		// --- АВТОБУС В ПРЕДЕЛАХ МАРШРУТА ---
		if state.IsOffRoute {
			// ТРИГГЕР: Автобус гулял, но ТОЛЬКО ЧТО ВЕРНУЛСЯ на трек
			duration := actualTime.Sub(state.OffRouteStartTime)
			durationMinutes := int(duration.Minutes())
			if durationMinutes < 0 {
				durationMinutes = 0
			}

			log.Printf("[SUCCESS] ✅ Автобус %s ВЕРНУЛСЯ на маршрут. Гулял: %d мин. Пиковое смещение: %.2f м",
				busCtx.BusNumber, durationMinutes, state.MaxDeviation)

			// ЖЕСТКАЯ ЕДИНСТВЕННАЯ ЗАПИСЬ В БАЗУ ЗА ВЕСЬ ИНЦИДЕНТ
			query := `
				INSERT INTO route_deviation_reports (
					city, route_number, bus_gov_number, off_route_start, off_route_end, 
					duration_minutes, max_deviation_meters, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

			_, err := db.Exec(query,
				busCtx.City,
				busCtx.RouteNumber,
				busCtx.BusNumber,
				state.OffRouteStartTime,
				actualTime,
				durationMinutes,
				state.MaxDeviation,
				actualTime,
			)
			if err != nil {
				log.Println("[DATABASE ERROR] Ошибка записи инцидента отклонения:", err)
			} else {
				log.Printf("[SUCCESS] Инцидент для автобуса %s успешно залогирован в БД.", busCtx.BusNumber)
			}

			// Сбрасываем флаги инцидента в памяти для следующего раза
			state.IsOffRoute = false
			state.MaxDeviation = 0
		}
	}
}
