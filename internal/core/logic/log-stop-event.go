package logic

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
	err := db.Get(&info, "SELECT route_number, route_number as bus_number FROM buses WHERE id=$1", busCtx.BusID)
	if err != nil {
		log.Printf("Ошибка получения инфо о автобусе из БД: %v. Используем данные из контекста.", err)
		info.RouteNumber = busCtx.RouteNumber
		info.BusNumber = busCtx.BusNumber
	}
	var delayMinutes int

	statusStr := "visited"
	stayStr := event.StayDuration.Round(time.Second).String()

	if event.IsSkipped {
		statusStr = "skipped"
		stayStr = "0s"
	}

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
		"00:00:00",
		event.ActualTime.Format("15:04:05"),
		stayStr,
		delayMinutes,
		statusStr,
		time.Now(),
	)

	if err != nil {
		log.Printf("НЕ удалось записать в БД: %v", err)
	} else {
		rows, _ := res.RowsAffected()
		log.Printf("Отчёт записан: %d строк", rows)
	}
}
