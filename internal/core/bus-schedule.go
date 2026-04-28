package core

import (
	"math"
	"orsavisionweb/internal/models"
	"time"
)

const (
	MaxEarlyMinutes float64 = 5.0
	MaxDelayMinutes float64 = 15.0
)

func CalculateDelay(event *models.StopEvent, schedule []string) {
	// Если расписания нет - выходим
	if len(schedule) == 0 {
		event.Status = "Нет расписания"
		return
	}

	var bestMatch time.Time
	minDiff := 1440.0 // Минут в сутках

	for _, tStr := range schedule {
		pTime, _ := time.Parse("15:04:05", tStr)
		// Привязываем время из расписания к текущей дате
		planned := time.Date(event.ActualTime.Year(), event.ActualTime.Month(), event.ActualTime.Day(),
			pTime.Hour(), pTime.Minute(), pTime.Second(), 0, event.ActualTime.Location())

		diff := math.Abs(event.ActualTime.Sub(planned).Minutes())
		if diff < minDiff {
			minDiff = diff
			bestMatch = planned
		}
	}

	// Считаем итоговую разницу (может быть отрицательной)
	finalDiff := event.ActualTime.Sub(bestMatch).Minutes()

	// ПРЯМАЯ ЗАПИСЬ В ПОЛЕ СТРУКТУРЫ
	if finalDiff < -MaxEarlyMinutes {
		event.Status = "Рано прибыл"
	} else if finalDiff > MaxDelayMinutes {
		event.Status = "Поздно прибыл"
	} else {
		event.Status = "Вовремя"
	}
}
