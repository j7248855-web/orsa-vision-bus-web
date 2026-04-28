package handler

import (
	"fmt"
	"net/http"
	"orsavisionweb/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/xuri/excelize/v2"
)

func GetOperationJourneyReport(ctx *gin.Context, conn *sqlx.DB) {
	busID := ctx.Param("bus_id")

	var reports []models.ReportRow

	// 1. Получаем данные
	query := `
		SELECT r.* FROM stop_reports r
		JOIN buses b ON r.bus_gov_number = b.bus_number
		WHERE b.id = $1
		ORDER BY r.created_at DESC`

	err := conn.Select(&reports, query, busID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить данные: " + err.Error()})
		return
	}

	// 2. Создаем Excel файл
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Отчет по остановкам"
	f.SetSheetName("Sheet1", sheetName)

	// Заголовки таблицы
	headers := []string{"ID", "Город", "Дата", "Маршрут", "Гос. номер", "Остановка", "План", "Факт", "Стоянка", "Задержка (мин)", "Статус"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 3. Заполняем данными
	for i, r := range reports {
		rowIdx := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIdx), r.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIdx), r.City)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIdx), r.ReportDate)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIdx), r.RouteNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIdx), r.BusGovNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIdx), r.StopName)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIdx), r.PlannedTime)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIdx), r.ActualTime)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowIdx), r.StayDuration)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIdx), r.DelayMinutes)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIdx), r.Status)
	}

	// 4. Настраиваем заголовки ответа для скачивания
	fileName := fmt.Sprintf("report_bus_%s_%s.xlsx", busID, time.Now().Format("2006-01-02_15-04"))

	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Cache-Control", "no-cache")

	// 5. Записываем файл в поток ответа Gin
	if err := f.Write(ctx.Writer); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при генерации файла: " + err.Error()})
	}
}
