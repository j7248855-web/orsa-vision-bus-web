package models

import "time"

type ReportRow struct {
	ID           int       `json:"id" db:"id"`
	City         string    `json:"city_name" db:"city"`
	ReportDate   string    `json:"report_date" db:"report_date"`
	RouteNumber  string    `json:"route_number" db:"route_number"`
	BusGovNumber string    `json:"bus_gov_number" db:"bus_gov_number"`
	StopName     string    `json:"stop_name" db:"stop_name"`
	PlannedTime  string    `json:"planned_time" db:"planned_time"`
	ActualTime   string    `json:"actual_time" db:"actual_time"`
	StayDuration string    `json:"stay_duration" db:"stay_duration"`
	DelayMinutes int       `json:"delay_minutes" db:"delay_minutes"`
	Status       string    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
