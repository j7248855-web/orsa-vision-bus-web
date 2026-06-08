package models

import "time"

// Тип устройства
type Device struct {
	ID       string `db:"id"`
	BusID    string `db:"bus_id"`
	DeviceIP string `json:"rtspLink" db:"device_ip"`
	Type     string `json:"type" db:"type"`     //камера это или телтоник
	Status   string `json:"status" db:"status"` //его статус в данный момент
}

// Основная структура автобуса
type Bus struct {
	ID             string        `json:"id" db:"id"`
	BusNumber      string        `json:"busNumber" db:"bus_number" binding:"required"` //Гос номер
	RouteNumber    string        `json:"rout" db:"route_number" binding:"required"`    //маршрут автобуса
	Status         string        `json:"status" db:"status" binding:"required"`        //статус для понятия активен
	City           string        `json:"city" db:"city" binding:"required"`
	SequenceNumber int           `json:"sequence_number" db:"sequence_number"`
	Devices        []Device      `json:"devices" db:"-"`
	Schedule       []ScheduleBus `json:"schedule"`
}

type ScheduleBus struct {
	SequenceNumber int       `json:"-" db:"sequence_number"`
	ArrivalTime    time.Time `json:"arrival_time" db:"arrival_time"`
	DepartuteTime  time.Time `json:"departure_time" db:"departure_time"`
}

type BusOnMap struct {
	BusNumber string  `json:"busNumber"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
}
