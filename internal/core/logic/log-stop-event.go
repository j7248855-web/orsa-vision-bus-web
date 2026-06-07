package logic

import (
	"log"
	"orsavisionweb/internal/models"
	"time"

	"github.com/jmoiron/sqlx"
)

func LogStopEvent(db *sqlx.DB, busCtx *models.BusContext, stop models.Stop, event *models.StopEvent) {
	log.Printf("[DB_DEBUG] Начало выполнения LogStopEvent для остановки: %s", stop.Name)

	var info struct {
		BusNumber   string `db:"bus_number"`
		RouteNumber string `db:"route_number"`
	}
	err := db.Get(&info, "SELECT route_number, route_number as bus_number FROM buses WHERE id=$1", busCtx.BusID)
	if err != nil {
		log.Printf("[DB_DEBUG] Ошибка получения инфо о автобусе из БД: %v. Используем данные из контекста.", err)
		info.RouteNumber = busCtx.RouteNumber
		info.BusNumber = busCtx.BusNumber
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
		log.Printf("[DB_DEBUG] Ошибка поиска планового времени: %v. Ставим дефолт.", err)
		plannedTime = "00:00:00"
	}

	var delayMinutes int
	if plannedTime != "00:00:00" && plannedTime != "--:--:--" {
		pTime, _ := time.Parse("15:04:05", plannedTime)
		plannedTotalMinutes := pTime.Hour()*60 + pTime.Minute()
		actualTotalMinutes := event.ActualTime.Hour()*60 + event.ActualTime.Minute()
		delayMinutes = actualTotalMinutes - plannedTotalMinutes
	}

	statusStr := "visited"
	stayStr := event.StayDuration.Round(time.Second).String()

	if event.IsSkipped {
		statusStr = "skipped"
		stayStr = "0s"
	}

	log.Printf("[DB_DEBUG] Подготовка инсерта: City=%s, Route=%s, GovNumber=%s, Status=%s", stop.City, info.RouteNumber, info.BusNumber, statusStr)

	query := `
        INSERT INTO stop_reports (
            city_name, report_date, route_number, trip_number, 
            bus_gov_number, stop_name, planned_time, actual_time, 
            stay_duration, delay_minutes, status, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	res, err := db.Exec(query,
		stop.City,
		event.ActualTime.Format("2006-01-02"),
		info.RouteNumber,
		busCtx.State.TripSequence,
		info.BusNumber,
		stop.Name,
		plannedTime,
		event.ActualTime.Format("15:04:05"),
		stayStr,
		delayMinutes,
		statusStr,
		time.Now(),
	)

	if err != nil {
		log.Printf("!!! КРИТИЧЕСКАЯ ОШИБКА ЗАПИСИ ОТЧЕТА В БД: %v", err)
	} else {
		rows, _ := res.RowsAffected()
		log.Printf("[DB_DEBUG] УСПЕШНО ЗАПИСАНО РЕПОРТОВ: %d строк", rows)
	}
}
