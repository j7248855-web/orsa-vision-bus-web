package models

import (
	"time"
)

type Dependence struct {
	FirtsSeenOnStation time.Time
	IsBusStop          bool
	LastPoint          []float64
	CurrentStopPoints  []float64
	LastTime           time.Time
	WasInRadius        bool
}
type BusContext struct {
	BusID       string       `db:"bus_id"`
	BusNumber   string       `db:"bus_number"` // Добавляем сюда ТС (000AAA)
	RouteNumber string       `db:"route_number"`
	Stop        []Stop       `db:"-"`
	Points      [][2]float64 `db:"-"`
	State       *Dependence  `db:"-"`
}
type StopEvent struct {
	StopID       int
	ActualTime   time.Time
	IsSkipped    bool //был ли на остановке
	StayDuration time.Duration
	Status       string
}
