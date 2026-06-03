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
	TripStatus         string    `json:"trip_status"`           // "idle" или "in_trip"
	CurrentStartStopID int       `json:"current_start_stop_id"` // ID конечной, откуда едет (или где стоит)
	PlanDeparture      string    `json:"plan_departure"`        // Плановое время выезда
	PlanArrival        string    `json:"plan_arrival"`          // Плановое время приезда
	ActualDeparture    time.Time `json:"actual_departure"`      // Реальное время выезда с конечной
}

// Информация про сам автобус
type BusContext struct {
	BusID          int          `db:"bus_id"`
	BusNumber      string       `db:"bus_number"` // Добавляем сюда ТС
	RouteNumber    string       `db:"route_number"`
	SequenceNumber int          `db:"sequence_number"`
	Stop           []Stop       `db:"-"`
	Points         [][2]float64 `db:"-"`
	State          *Dependence  `db:"-"`
}

// Модель чтобы знать нарушения на остановках
type StopEvent struct {
	StopID       int
	ActualTime   time.Time
	IsSkipped    bool //был ли на остановке
	StayDuration time.Duration
	Status       string
}

// Для того знать насколько идёт отклонение от привычного маршрута
type DeviationResult struct {
	IsOffRoute bool
	Value      string
}
