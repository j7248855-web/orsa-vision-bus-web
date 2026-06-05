package models

import "time"

type ReportRow struct {
	ID           int       `json:"id" db:"id"`
	City         string    `json:"city_name" db:"city"`
	ReportDate   string    `json:"report_date" db:"report_date"`
	RouteNumber  string    `json:"route_number" db:"route_number"`
	BusGovNumber string    `json:"bus_gov_number" db:"bus_gov_number"`
	TripSequence int       `json:"trip_sequence" db:"trip_sequence"`
	FromStopName string    `json:"from_stop_name" db:"from_stop_name"`
	ToStopName   string    `json:"to_stop_name" db:"to_stop_name"`
	PlannedTime  string    `json:"planned_time" db:"planned_time"`
	ActualTime   string    `json:"actual_time" db:"actual_time"`
	DurationFact string    `json:"duration_fact" db:"duration_fact"`
	DelayMinutes int       `json:"delay_minutes" db:"delay_minutes"`
	Status       string    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type ReportRequest struct {
	ReportType string `json:"report_type"`
	City       string `json:"city"`
	BusID      int    `json:"bus_id"`
	DateFrom   string `json:"date_from"`
	DateTo     string `json:"date_to"`
}

type ReportListItem struct {
	ReportID   string `json:"report_id"`
	ReportName string `json:"report_name"`
	ReportType string `json:"report_type"`
	BusID      int    `json:"bus_id"`
	DateRange  string `json:"date_range"`
}
