package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/xuri/excelize/v2"
)

func GenerateViolationsExcel(ctx *gin.Context, db *sqlx.DB) (*excelize.File, error) {
	busID := ctx.Query("bus_id")
	if busID == "" {
		return nil, fmt.Errorf("bus_id is required")
	}

	rows, err := db.Queryx(`
        SELECT created_at, route_num, plate_num, violation_type, value 
        FROM bus_violations 
        WHERE bus_id = $1
        ORDER BY created_at DESC
    `, busID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	f := excelize.NewFile()
	sheet := "Нарушения"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"Дата/Время", "Маршрут", "Госномер", "Тип нарушения", "Значение"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	rowIdx := 2
	for rows.Next() {
		var r struct {
			CreatedAt time.Time `db:"created_at"`
			Route     string    `db:"route_num"`
			Plate     string    `db:"plate_num"`
			VType     string    `db:"violation_type"`
			Value     string    `db:"value"`
		}
		if err := rows.StructScan(&r); err != nil {
			continue
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), r.CreatedAt.Format("02.01.2006 15:04:05"))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), r.Route)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), r.Plate)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), r.VType)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), r.Value)
		rowIdx++
	}

	f.SetColWidth(sheet, "A", "A", 20)
	f.SetColWidth(sheet, "D", "E", 25)

	return f, nil
}
