package reports

import (
	"fmt"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/xuri/excelize/v2"
)

func GetOperationJourneyReport(ctx *gin.Context, db *sqlx.DB, req models.ReportRequest) (*excelize.File, error) {
	query := `
		SELECT r.id, r.city, r.report_date, r.route_number, r.bus_gov_number, r.trip_sequence,
		       r.from_stop_name, r.to_stop_name, r.planned_time, r.actual_time,
		       r.duration_fact, r.delay_minutes, r.status, r.created_at
		FROM trip_reports r
		JOIN buses b ON r.bus_gov_number = b.bus_number
		WHERE b.id = $1`

	var args []interface{}
	args = append(args, req.BusID)
	argCount := 2

	// Динамически докидываем фильтр дат, если фронт их передал
	if req.DateFrom != "" {
		query += fmt.Sprintf(" AND r.created_at >= $%d::timestamp", argCount)
		args = append(args, req.DateFrom+" 00:00:00")
		argCount++
	}
	if req.DateTo != "" {
		query += fmt.Sprintf(" AND r.created_at <= $%d::timestamp", argCount)
		args = append(args, req.DateTo+" 23:59:59")
		argCount++
	}

	query += " ORDER BY r.created_at DESC"
	var reports []models.ReportRow
	err := db.SelectContext(ctx, &reports, query, args...)
	if err != nil {
		return nil, err
	}
	f := excelize.NewFile()
	sheetName := "Оперативный отчет по рейсам"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{
		"ID", "Город", "Дата отчета", "Маршрут", "Гос. номер", "Рейс №",
		"Откуда", "Куда", "План время", "Факт время",
		"Время в пути", "Задержка (мин)", "Статус",
	}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	for i, r := range reports {
		rowIdx := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIdx), r.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIdx), r.City)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIdx), r.ReportDate)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIdx), r.RouteNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIdx), r.BusGovNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIdx), r.TripSequence)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIdx), r.FromStopName)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIdx), r.ToStopName)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowIdx), r.PlannedTime)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIdx), r.ActualTime)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIdx), r.DurationFact)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIdx), r.DelayMinutes)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowIdx), r.Status)
	}

	return f, nil
}
