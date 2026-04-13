package models

import "time"

type Stop struct {
	ID          int        `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Position    [2]float64 `json:"position" db:"-"`    //для JSON
	Lat         float64    `json:"-" db:"lat"`         //для БД
	Lon         float64    `json:"-" db:"lng"`         //для БД
	Radius      float64    `json:"radius" db:"radius"` //может быть nil
	Type        string     `json:"type" db:"type"`
	Azimuth     float64    `json:"azimuth" db:"azimuth"`
	City        string     `json:"city" db:"city"`
	PlannedTime time.Time  `json:"plannedTime,omitempty" db:"planned_time"`
}

type Route struct {
	ID      string       `json:"id" db:"id"`
	RouteID string       `json:"routeNumber" db:"route_number"`
	Name    string       `json:"name" db:"name"`
	City    string       `json:"city" db:"-"`
	Points  [][2]float64 `json:"points" db:"-"`
	Stops   []Stop       `json:"stops" db:"-"`
	Status  string       `json:"status" db:"status"`
}

// Отдельная структура для точек пути
type RoutePoint struct {
	RouteID string  `db:"route_id"`
	Lat     float64 `db:"lat"`
	Lon     float64 `db:"lng"`
}
