package core

import (
	"context"
	"encoding/json"
	"log"
	"orsavisionweb/internal/models"
	"time"

	"github.com/jmoiron/sqlx"
)

type IncomingAlarm struct {
	DeviceIP string `json:"device_ip"`
}

func BusEmergency(data []byte, conn *sqlx.DB) {
	var alarm IncomingAlarm
	var alert models.EmergencyInformation

	err := json.Unmarshal(data, &alarm)
	if err != nil {
		log.Println("Не удалось распарсить данные в структуру")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	findQuery := `
		SELECT b.id as bus_id, b.bus_number as bus_number, b.route_number as route_number
		FROM devices d
		JOIN buses b ON d.bus_id = b.id
		WHERE d.device_ip = $1 AND d.type = 'teltonic'
		LIMIT 1
	`
	err = conn.GetContext(ctx, &alert, findQuery, alarm.DeviceIP)
	if err != nil {
		log.Println("Автобус с IP трекера не найден в системе", err)
		return
	}

	insertQuery := `
		INSERT INTO buses_emergencies (bus_id, bus_number, route_number, emergency_at)
		VALUES ($1, $2, $3, $4)
	`

	currentTime := time.Now()

	_, err = conn.ExecContext(ctx, insertQuery, alert.BusID, alert.BusNumber, alert.RouteNumber, currentTime)
	if err != nil {
		log.Println("Не удалось записать тревогу в базу", err)
		return
	}
}
