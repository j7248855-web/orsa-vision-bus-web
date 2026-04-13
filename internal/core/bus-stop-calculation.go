package core

import (
	"log"
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
	dLon := busPos[0] - stopPos[0]
	dLat := busPos[1] - stopPos[1]
	radian := busPos[1] * math.Pi / 180

	dFb := dLat * 111111.0
	dAb := (dLon * 111111.0) * math.Cos(radian)
	dist := math.Sqrt(math.Pow(dFb, 2) + math.Pow(dAb, 2))

	return dist <= radius
}

func CalculateStopStation(d *models.Dependence, busPos []float64, lastBusPos []float64, timeDiff time.Duration, stopPos []float64, stopRadius float64, actualTime time.Time, busCourse float64, stopAzimuth float64) {
	//Вычисление направления азимута остановки и автобуса (смотрят ли они в одну сторону)
	angleDiff := math.Mod(math.Abs(busCourse-stopAzimuth), 360)
	if angleDiff > 180 {
		angleDiff = 360 - angleDiff
	}
	//Если больше 90 то не в нашу сторону едет
	if angleDiff > 90 {
		return
	}
	speed := SpeedCalculation(lastBusPos, busPos, timeDiff)
	inRadius := RadiusCalculation(busPos, stopPos, stopRadius)

	if inRadius && speed <= 7 {
		if d.FirtsSeenOnStation.IsZero() {
			d.FirtsSeenOnStation = actualTime
			return
		}
	}

	if inRadius && speed <= 10 && !d.FirtsSeenOnStation.IsZero() {
		if time.Since(d.FirtsSeenOnStation) >= 10*time.Second {
			log.Println("Остановка засчитана!")
			d.IsBusStop = true
			d.FirtsSeenOnStation = time.Time{}
		}
		return
	}

	if !inRadius || speed > 10 {
		d.FirtsSeenOnStation = time.Time{}
		log.Println("Автобус проехал остановку") //изменить на Excel
	}
}
