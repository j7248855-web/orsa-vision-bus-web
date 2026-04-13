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
}
type BusContext struct {
	BusID       string       `db:"bus_id"`
	RouteNumber string       `db:"route_number"`
	Stop        []Stop       `db:"-"`
	Points      [][2]float64 `db:"-"`
	State       *Dependence  `db:"-"`
}
