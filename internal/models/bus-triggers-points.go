package models

import "time"

type StopPoint struct {
	StopID      string    `json:"stop_id"`
	Name        string    `json:"name"`
	Radius      int       `json:"radius"`
	BusStopTime time.Time `json:"bus-stop-time"`
}
