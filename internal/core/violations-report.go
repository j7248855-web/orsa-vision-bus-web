package core

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/jmoiron/sqlx"
)

func ViolationsReport(db *sqlx.DB, busCtx *models.BusContext, vType, value string) {
	var info struct {
		BusNumber   string `db:"bus_number"`
		RouteNumber string `db:"route_number"`
	}

	err := db.Get(&info, "SELECT bus_number, route_number FROM buses WHERE id=$1", busCtx.BusID)
	if err != nil {
		log.Printf("Ошибка получения данных автобуса для отчета: %v", err)
		return
	}

	query := `INSERT INTO bus_violations (bus_id, route_num, plate_num, violation_type, value) 
              VALUES ($1, $2, $3, $4, $5)`

	_, err = db.Exec(query, busCtx.BusID, info.RouteNumber, info.BusNumber, vType, value)
	if err != nil {
		log.Printf("Ошибка записи нарушения в БД: %v", err)
	}
}
