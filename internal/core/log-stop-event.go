package core

import (
	"log"
	"orsavisionweb/internal/models"
	"time"

	"github.com/jmoiron/sqlx"
)

func LogStopEvent(db *sqlx.DB, busCtx *models.BusContext, stop models.Stop, event *models.StopEvent) {
	var info struct {
		BusNumber   string `db:"bus_number"`
		RouteNumber string `db:"route_number"`
	}
	err := db.Get(&info, "SELECT bus_number, route_number FROM buses WHERE id=$1", busCtx.BusID)
	if err != nil {
		log.Printf("Ошибка получения данных автобуса: %v", err)
		return
	}
	var plannedTime string
	err = db.Get(&plannedTime, `
        SELECT arrival_time 
        FROM stop_schedules 
        WHERE bus_id = $1 AND stop_id = $2 
        ORDER BY ABS(EXTRACT(EPOCH FROM (arrival_time::time - $3::time))) 
        LIMIT 1`,
		busCtx.BusID, stop.ID, event.ActualTime.Format("15:04:05"))

	if err != nil {
		log.Printf("Расписание не найдено для автобуса %s на остановке %d", busCtx.BusID, stop.ID)
		plannedTime = "--:--:--"
	}
	//Вычисление разницы во времени
	var delayMinutes int
	if plannedTime != "--:--:--" {
		pTime, _ := time.Parse("15:04:05", plannedTime)
		plannedTotalMinutes := pTime.Hour()*60 + pTime.Minute()
		actualTotalMinutes := event.ActualTime.Hour()*60 + event.ActualTime.Minute()
		delayMinutes = actualTotalMinutes - plannedTotalMinutes
	}
	stayStr := event.StayDuration.Round(time.Second).String()
	_, err = db.Exec(`
        INSERT INTO stop_reports (
            report_date, 
            route_number, 
            bus_gov_number, 
            stop_name, 
            planned_time, 
            actual_time, 
            stay_duration, 
            delay_minutes, 
            status, 
            city
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		event.ActualTime.Format("2006-01-02"), // Дата
		info.RouteNumber,                      // Маршрут
		info.BusNumber,                        // ТС
		stop.Name,                             // Остановка (берем из структуры)
		plannedTime,                           // План
		event.ActualTime.Format("15:04:05"),   // Факт
		stayStr,                               // Тут будет расчет стоянки (time.Since)
		delayMinutes,                          // Отклонение
		event.Status,                          // Статус
		stop.City,                             // Город (берем из таблицы stops)
	)

	if err != nil {
		log.Printf("Ошибка записи отчета: %v", err)
	}
}
