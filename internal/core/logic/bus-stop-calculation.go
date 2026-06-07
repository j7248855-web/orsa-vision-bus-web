package logic

import (
	"fmt"
	"math"
	"orsavisionweb/internal/models"
	"time"
)

// Ключ теперь будет строкой: "busID_stopID", чтобы автобусы не мешали друг другу
var (
	stopArrivals = make(map[string]time.Time)
	wasInRadius  = make(map[string]bool)
)

func SpeedCalculation(p1, p2 []float64, duration time.Duration) float64 {
	if duration <= 0 {
		return 0
	}
	radian := p1[0] * math.Pi / 180
	dLat := p2[0] - p1[0]
	dLon := p2[1] - p1[1]
	X := dLat * 111111.0
	Y := (dLon * 111111.0) * math.Cos(radian)
	D := math.Sqrt(math.Pow(X, 2) + math.Pow(Y, 2))
	return (D / duration.Seconds()) * 3.6
}

func RadiusCalculation(busPos, stopPos []float64, radius float64) bool {
	dLat := busPos[0] - stopPos[0]
	dLon := busPos[1] - stopPos[1]
	radian := busPos[0] * math.Pi / 180

	dFb := dLat * 111111.0
	dAb := (dLon * 111111.0) * math.Cos(radian)
	dist := math.Sqrt(math.Pow(dFb, 2) + math.Pow(dAb, 2))

	return dist <= radius
}

func CalculateStopStation(d *models.Dependence, busPos []float64, lastBusPos []float64, timeDiff time.Duration, stopPos []float64, stopRadius float64, actualTime time.Time, busCourse float64, stopAzimuth float64, stopID int, busID int) *models.StopEvent {

	// Генерируем уникальный ключ для мапы в памяти
	mapKey := fmt.Sprintf("%d_%d", busID, stopID)

	// Проверка азимута направления
	angleDiff := math.Mod(math.Abs(busCourse-stopAzimuth), 360)
	if angleDiff > 180 {
		angleDiff = 360 - angleDiff
	}
	if angleDiff > 90 {
		return nil
	}

	speed := SpeedCalculation(lastBusPos, busPos, timeDiff)
	inRadius := RadiusCalculation(busPos, stopPos, stopRadius)

	// Если заехал в радиус остановки
	if inRadius {
		wasInRadius[mapKey] = true

		// Если скорость низкая (автобус встал), фиксируем время прибытия
		if speed <= 10 && stopArrivals[mapKey].IsZero() {
			stopArrivals[mapKey] = actualTime
		}
	}

	// Если автобус НЕ в радиусе (выехал из него)
	if !inRadius {
		if wasInRadius[mapKey] {
			entryTime := stopArrivals[mapKey]

			// Вариант А: Автобус стоял на остановке (скорость падала <= 10) и постоял от 5 секунд
			if !entryTime.IsZero() && actualTime.Sub(entryTime) >= 5*time.Second {
				duration := actualTime.Sub(entryTime)

				event := &models.StopEvent{
					ActualTime:   actualTime,
					IsSkipped:    false,
					StayDuration: duration,
				}

				// Очищаем кэш для этого автобуса на этой остановке
				delete(stopArrivals, mapKey)
				delete(wasInRadius, mapKey)
				return event
			}

			// Вариант Б: Автобус БЫЛ в радиусе, но вылетел из него, так и не остановившись (Пропуск!)
			event := &models.StopEvent{
				ActualTime:   actualTime,
				IsSkipped:    true, // <--- ВОТ ОН, ПРОПУСК!
				StayDuration: 0,
			}

			// Очищаем кэш
			delete(stopArrivals, mapKey)
			delete(wasInRadius, mapKey)
			return event
		}
	}

	return nil
}
