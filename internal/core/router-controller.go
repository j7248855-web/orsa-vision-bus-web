package core

import (
	"log"
	"math"
)

const (
	MetersPerDegree = 111111.0
)

func CheckDeviation(lat, lon float64, routePoints [][2]float64) bool {
	log.Printf("DEBUG: BusLat=%f, BusLon=%f | RouteLat=%f, RouteLon=%f",
		lat, lon, routePoints[0][0], routePoints[0][1])
	if len(routePoints) < 2 {
		return false
	}
	var minDeviation = math.MaxFloat64
	busPos := []float64{lat, lon}
	// Итерируемся по сегментам маршрута (от точки к точке)
	for i := 0; i < len(routePoints)-1; i++ {
		p1 := routePoints[i]
		p2 := routePoints[i+1]

		vecRoad := []float64{p2[0] - p1[0], p2[1] - p1[1]}
		vecToBus := []float64{busPos[0] - p1[0], busPos[1] - p1[1]}

		dotProduct := (vecToBus[0] * vecRoad[0]) + (vecToBus[1] * vecRoad[1])
		magRoadSq := (vecRoad[0] * vecRoad[0]) + (vecRoad[1] * vecRoad[1])

		if magRoadSq == 0 {
			continue
		}

		t := dotProduct / magRoadSq

		// Ограничиваем t, чтобы проекция не улетала за пределы отрезка дороги
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}

		projection := []float64{
			p1[0] + (vecRoad[0] * t),
			p1[1] + (vecRoad[1] * t),
		}
		deltaLon := busPos[0] - projection[0]
		deltaLat := busPos[1] - projection[1]

		metersY := deltaLat * MetersPerDegree
		busRad := busPos[1] * math.Pi / 180.0
		metersX := deltaLon * MetersPerDegree * math.Cos(busRad)
		currentDev := math.Sqrt(math.Pow(metersX, 2) + math.Pow(metersY, 2))
		if currentDev < minDeviation {
			minDeviation = currentDev
		}
	}
	if minDeviation > 30.0 && minDeviation != math.MaxFloat64 {
		log.Printf("Автобус сместился на: %.2f метра", minDeviation)
		return true
	}
	return false
}
