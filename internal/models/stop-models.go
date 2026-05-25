package models

import "time"

type StopPoint struct {
	StopID      string    `json:"stop_id" binding:"required"`
	Name        string    `json:"name"  binding:"required"`
	Radius      int       `json:"radius"  binding:"required"`
	BusStopTime time.Time `json:"bus-stop-time"  binding:"required"`
}
