package models

// Тип устройства
type Device struct {
	ID       string `json:"id" db:"id"`
	BusID    string `db:"bus_id"`
	DeviceIP string `json:"rtspLink" db:"device_ip"`
	Type     string `json:"type" db:"type"`     //камера это или телтоник
	Status   string `json:"status" db:"status"` //его статус в данный момент
}

// Основная структура автобуса
type Bus struct {
	ID          string   `json:"id" db:"id" binding:"required"`
	BusNumber   string   `json:"busNumber" db:"bus_number" binding:"required"` //Гос номер
	RouteNumber string   `json:"rout" db:"route_number" binding:"required"`    //маршрут автобуса
	Status      string   `json:"status" db:"status" binding:"required"`        //статус для понятия активен
	City        string   `json:"city" db:"city" binding:"required"`
	Devices     []Device `json:"devices" db:"-"`
}
