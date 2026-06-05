package logic

import (
	"math"
	"orsavisionweb/internal/models"
	"time"
)

// Хранилища в памяти прямо внутри пакета logic, чтобы не трогать твои structs в моделях
var (
	stopArrivals = make(map[int]time.Time) // Когда въехал на остановку [stopID] -> Time
	wasInRadius  = make(map[int]bool)      // Был ли в радиусе [stopID] -> bool
)

func SpeedCalculation(p1, p2 []float64, duration time.Duration) float64 {
	if duration <= 0 {
		return 0
	}
	radian := p1[0] * math.Pi / 180 // Индекс 0 - это Lat
	dLat := p2[0] - p1[0]
	dLon := p2[1] - p1[1]
	X := dLat * 111111.0
	Y := (dLon * 111111.0) * math.Cos(radian)
	D := math.Sqrt(math.Pow(X, 2) + math.Pow(Y, 2))
	return (D / duration.Seconds()) * 3.6
}

func RadiusCalculation(busPos, stopPos []float64, radius float64) bool {
	// Индексы: [0] - Lat, [1] - Lon
	dLat := busPos[0] - stopPos[0]
	dLon := busPos[1] - stopPos[1]
	radian := busPos[0] * math.Pi / 180

	dFb := dLat * 111111.0
	dAb := (dLon * 111111.0) * math.Cos(radian)
	dist := math.Sqrt(math.Pow(dFb, 2) + math.Pow(dAb, 2))

	return dist <= radius
}

// Добавили stopID в конец аргументов, чтобы изолировать логику
func CalculateStopStation(d *models.Dependence, busPos []float64, lastBusPos []float64, timeDiff time.Duration, stopPos []float64, stopRadius float64, actualTime time.Time, busCourse float64, stopAzimuth float64, stopID int) *models.StopEvent {
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
		wasInRadius[stopID] = true

		// Если время первого появления на ЭТОЙ остановке еще не записано и скорость низкая
		if speed <= 10 && stopArrivals[stopID].IsZero() {
			stopArrivals[stopID] = actualTime
		}
	}

	// Если автобус НЕ в радиусе (выехал из него или еще не доехал)
	if !inRadius {
		// Проверяем, был ли он внутри этой конкретной остановки до этого
		if wasInRadius[stopID] {
			entryTime := stopArrivals[stopID]

			// Если он там реально постоял хотя бы 5 секунд по GPS времени
			if !entryTime.IsZero() && actualTime.Sub(entryTime) >= 5*time.Second {
				duration := actualTime.Sub(entryTime)

				event := &models.StopEvent{
					ActualTime:   actualTime,
					IsSkipped:    false,
					StayDuration: duration,
				}

				// Очищаем данные только для этой остановки
				stopArrivals[stopID] = time.Time{}
				wasInRadius[stopID] = false
				return event
			}

			// Если просто пролетел мимо (был в радиусе, но не стоял) — сбрасываем флаг
			wasInRadius[stopID] = false
			stopArrivals[stopID] = time.Time{}
		}
	}

	return nil
}
