package models

import "time"

type EmergencyInformation struct {
	BusID       int       `json:"bus_id" db:"bus_id"`
	BusNumber   string    `json:"bus_number" db:"bus_number"`
	RouteNumber string    `json:"route_number" db:"route_number"`
	EmergencyAt time.Time `json:"emergency_at" db:"emergency_at"`
}
