package logic

import (
	"math"
	"orsavisionweb/internal/models"
	"time"
)

// Теперь принимаем две точки и время между ними
func SpeedCalculation(p1, p2 []float64, duration time.Duration) float64 {
	radian := p1[1] * math.Pi / 180
	dF := p2[1] - p1[1]
	dA := p2[0] - p1[0]
	X := dF * 111111.0
	Y := (dA * 111111.0) * math.Cos(radian)
	D := math.Sqrt(math.Pow(X, 2) + math.Pow(Y, 2))

	dT := duration.Seconds()
	if dT <= 0 {
		return 0
	} // Защита от деления на ноль
	dS := D / dT
	return dS * 3.6
}

// Принимаем текущую позицию автобуса и позицию целевой остановки
func RadiusCalculation(busPos, stopPos []float64, radius float64) bool {
	busLat := busPos[0]
	busLon := busPos[1]
	stopLat := stopPos[0]
	stopLon := stopPos[1]

	radian := busLat * math.Pi / 180.0
	deltaLat := busLat - stopLat
	deltaLon := busLon - stopLon

	Y := deltaLat * 111111.0
	X := deltaLon * 111111.0 * math.Cos(radian)
	distance := math.Sqrt(math.Pow(X, 2) + math.Pow(Y, 2))

	return distance <= radius
}

func CalculateStopStation(d *models.Dependence, busPos []float64, lastBusPos []float64, timeDiff time.Duration, stopPos []float64, stopRadius float64, actualTime time.Time, busCourse float64, stopAzimuth float64) *models.StopEvent {
	angleDiff := math.Mod(math.Abs(busCourse-stopAzimuth), 360)
	if angleDiff > 180 {
		angleDiff = 360 - angleDiff
	}
	if angleDiff > 90 {
		return nil
	}

	inRadius := RadiusCalculation(busPos, stopPos, stopRadius)

	if inRadius {
		if !d.WasInRadius {
			d.WasInRadius = true
			d.FirtsSeenOnStation = actualTime
		}
	} else {
		if d.WasInRadius {
			duration := actualTime.Sub(d.FirtsSeenOnStation)
			var event *models.StopEvent

			if duration >= 15*time.Second {
				event = &models.StopEvent{
					ActualTime:   actualTime,
					IsSkipped:    false,
					StayDuration: duration,
				}
			} else {
				event = &models.StopEvent{
					ActualTime:   actualTime,
					IsSkipped:    true,
					StayDuration: 0,
				}
			}

			d.FirtsSeenOnStation = time.Time{}
			d.WasInRadius = false
			return event
		}
	}
	return nil
}
