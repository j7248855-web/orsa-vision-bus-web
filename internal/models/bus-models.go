package models

// Тип устройства
type Device struct {
	ID       string `json:"-" db:"id"`
	RTSPlink string `json:"rtspLink" db:"rtsp_link"`
	Type     string `json:"type" db:"type"`     //камера это или телтоник
	Status   string `json:"status" db:"status"` //его статус в данный момент
}

// Основная структура автобуса
type Bus struct {
	ID          string   `json:"id" db:"id"`
	BusNumber   string   `json:"busNumber" db:"bus_number"` //Гос номер
	RouteNumber string   `json:"rout" db:"route_number"`    // 12, 36, 503
	Status      string   `json:"status" db:"status"`        // такой же статус
	Devices     []Device `json:"devices" db:"-"`
}
