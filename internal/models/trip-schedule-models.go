package models

type TripSteps struct {
	RouteID        int    `db:"route_id"`
	SequenceNumber int    `db:"sequence_number" json:"sequence_number"`
	DepartureTime  string `db:"departure_time" json:"departure_time"`
	ArrivalTime    string `db:"arrival_time"   json:"arrival_time"`
}

// Полный график всех автобусов на одном маршруте
type Schedule struct {
	RouteID int `json:"route_id"`
	Trips   [][]TripSteps
}
