package core

import (
	"log"
	"time"
)

const (
	MaxEarlyMinutes float64 = 5.0
	MaxDelayMinutes float64 = 15.0
)

func CalculateDelay(actualTime time.Time, plannedTime time.Time) {

	diff := actualTime.Sub(plannedTime).Minutes()
	if diff < -MaxEarlyMinutes {
		log.Println("Слишком раннее прибытие") //test
		return
	}
	if diff > MaxDelayMinutes {
		log.Println("Слишком позднее прибытие") //test
		return
	}
	log.Println("Прибыл по расписанию") //test
}
