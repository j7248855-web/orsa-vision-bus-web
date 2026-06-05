package reports

import (
	"fmt"
	"log"
	"orsavisionweb/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/xuri/excelize/v2"
)

func HandleGenerateReport(ctx *gin.Context, conn *sqlx.DB) {
	var req models.ReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "Неверный формат запроса", "details": err.Error()})
		return
	}

	var file *excelize.File
	var err error
	var defaultName string

	// Переключаемся в зависимости от того, какой тип отчета выбрал пользователь на фронте
	switch req.ReportType {
	case "operational":
		defaultName = "operational_report"
		file, err = GetOperationJourneyReport(ctx, conn, req)
	case "violations_trip":
		defaultName = "route_deviations_report"
		// Передаем в функцию строку-фильтр для базы
		file, err = GenerateViolationsExcelFiltered(ctx, conn, req, "Выход с маршрута")
	case "violations_stops":
		defaultName = "skipped_stops_report"
		// Передаем в функцию строку-фильтр для базы
		file, err = GenerateViolationsExcelFiltered(ctx, conn, req, "Пропуск остановки")
	default:
		ctx.JSON(400, gin.H{"error": "Неизвестный вид отчетности"})
		return
	}

	if err != nil {
		ctx.JSON(500, gin.H{"error": "Ошибка генерации отчета: " + err.Error()})
		return
	}
	defer file.Close()

	fileName := fmt.Sprintf("%s_bus_%d_%s.xlsx", defaultName, req.BusID, time.Now().Format("2006-01-02_15-04"))

	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Cache-Control", "no-cache")

	if err := file.Write(ctx.Writer); err != nil {
		log.Println("Ошибка записи файла в стрим:", err)
	}
}

// Отдача всех данных на фронт по отчётам
func HandleGetReportsList(ctx *gin.Context, conn *sqlx.DB) {
	var req models.ReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "Неверный формат запроса", "details": err.Error()})
		return
	}

	dateRange := "За всё время"
	if req.DateFrom != "" && req.DateTo != "" {
		dateRange = fmt.Sprintf("с %s по %s", req.DateFrom, req.DateTo)
	}

	var list []models.ReportListItem

	switch req.ReportType {
	case "operational":
		list = append(list, models.ReportListItem{
			ReportID:   fmt.Sprintf("op_%d_%s", req.BusID, time.Now().Format("20060102")),
			ReportName: fmt.Sprintf("Оперативный отчет по рейсам (Автобус ID %d)", req.BusID),
			ReportType: "operational",
			BusID:      req.BusID,
			DateRange:  dateRange,
		})
	case "violations_trip":
		list = append(list, models.ReportListItem{
			ReportID:   fmt.Sprintf("viol_trip_%d_%s", req.BusID, time.Now().Format("20060102")),
			ReportName: fmt.Sprintf("Отчет по нарушениям: Выход с маршрута (Автобус ID %d)", req.BusID),
			ReportType: "violations_trip",
			BusID:      req.BusID,
			DateRange:  dateRange,
		})
	case "violations_stops":
		list = append(list, models.ReportListItem{
			ReportID:   fmt.Sprintf("viol_stops_%d_%s", req.BusID, time.Now().Format("20060102")),
			ReportName: fmt.Sprintf("Отчет по нарушениям: Пропуск остановки (Автобус ID %d)", req.BusID),
			ReportType: "violations_stops",
			BusID:      req.BusID,
			DateRange:  dateRange,
		})
	case "all":
		list = append(list, []models.ReportListItem{
			{
				ReportID:   fmt.Sprintf("op_%d_%s", req.BusID, time.Now().Format("20060102")),
				ReportName: fmt.Sprintf("Оперативный отчет по рейсам (Автобус ID %d)", req.BusID),
				ReportType: "operational",
				BusID:      req.BusID,
				DateRange:  dateRange,
			},
			{
				ReportID:   fmt.Sprintf("viol_trip_%d_%s", req.BusID, time.Now().Format("20060102")),
				ReportName: fmt.Sprintf("Отчет по нарушениям: Выход с маршрута (Автобус ID %d)", req.BusID),
				ReportType: "violations_trip",
				BusID:      req.BusID,
				DateRange:  dateRange,
			},
			{
				ReportID:   fmt.Sprintf("viol_stops_%d_%s", req.BusID, time.Now().Format("20060102")),
				ReportName: fmt.Sprintf("Отчет по нарушениям: Пропуск остановки (Автобус ID %d)", req.BusID),
				ReportType: "violations_stops",
				BusID:      req.BusID,
				DateRange:  dateRange,
			},
		}...)
	default:
		ctx.JSON(400, gin.H{"error": "Неизвестный вид отчетности"})
		return
	}

	ctx.JSON(200, list)
}
