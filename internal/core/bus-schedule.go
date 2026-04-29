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

func CalculateDelay(event *models.StopEvent, schedule []string) int {
	if len(schedule) == 0 {
		event.Status = "Нет расписания"
		return 0
	}

	var bestMatch time.Time
	minDiff := 1440.0

	for _, tStr := range schedule {
		// УНИВЕРСАЛЬНЫЙ ПАРСИНГ (на случай если секунд нет)
		layout := "15:04:05"
		if len(tStr) == 5 {
			layout = "15:04"
		}

		pTime, err := time.Parse(layout, tStr)
		if err != nil {
			continue
		}

		planned := time.Date(event.ActualTime.Year(), event.ActualTime.Month(), event.ActualTime.Day(),
			pTime.Hour(), pTime.Minute(), pTime.Second(), 0, event.ActualTime.Location())

		diff := math.Abs(event.ActualTime.Sub(planned).Minutes())
		if diff < minDiff {
			minDiff = diff
			bestMatch = planned
		}
	}

	// Итоговая разница в минутах
	finalDiff := int(event.ActualTime.Sub(bestMatch).Minutes())

	// Пишем статус
	if float64(finalDiff) < -MaxEarlyMinutes {
		event.Status = "Рано прибыл"
	} else if float64(finalDiff) > MaxDelayMinutes {
		event.Status = "Поздно прибыл"
	} else {
		event.Status = "Вовремя"
	}

	return finalDiff // Теперь функция ВОЗВРАЩАЕТ число
}
