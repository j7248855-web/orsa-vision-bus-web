package core

import (
	"encoding/csv"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func ParsingScheduleCSV(ctx *gin.Context, conn *sqlx.DB) {
	file, err := ctx.FormFile("filecsv")
	if err != nil {
		ctx.JSON(400, gin.H{"status": "Файл не получен"})
		return
	}

	openFile, err := file.Open()
	if err != nil {
		ctx.JSON(400, gin.H{"status": "Не удалось открыть файл"})
		return
	}
	defer openFile.Close()

	reader := csv.NewReader(openFile)
	records, err := reader.ReadAll()
	if err != nil {
		log.Println("Не удалось прочитать файл:", err)
		ctx.JSON(400, gin.H{"status": "Ошибка чтения CSV"})
		return
	}

	if len(records) < 2 {
		ctx.JSON(400, gin.H{"status": "Файл пустой"})
		return
	}

	headerMap := make(map[string]int)
	for i, name := range records[0] {
		cleanName := strings.Trim(name, " \ufeff\"")
		headerMap[cleanName] = i
	}

	requiredColumns := []string{"bus_id", "stop_id", "arrival_time", "day_type"}
	for _, col := range requiredColumns {
		if _, exists := headerMap[col]; !exists {
			log.Println("В файле нет колонки:", col)
			ctx.JSON(400, gin.H{"status": "В файле нет колонки: " + col})
			return
		}
	}

	tx := conn.MustBegin()
	for i := 1; i < len(records); i++ {
		row := records[i]

		if len(row) == 0 || row[headerMap["bus_id"]] == "" {
			continue
		}

		busID := strings.Trim(row[headerMap["bus_id"]], " \"")
		stopID := strings.Trim(row[headerMap["stop_id"]], " \"")
		arrivalTime := strings.Trim(row[headerMap["arrival_time"]], " \"")
		dayType := strings.Trim(row[headerMap["day_type"]], " \"")

		if busID == "" || stopID == "" {
			continue
		}

		_, err := tx.Exec(`
			INSERT INTO stop_schedules (bus_id, stop_id, arrival_time, day_type) 
			VALUES ($1, $2, $3, $4)`,
			busID, stopID, arrivalTime, dayType,
		)

		if err != nil {
			tx.Rollback()
			log.Printf("Ошибка на строке %d: %v", i+1, err)
			ctx.JSON(500, gin.H{"status": "Ошибка БД"})
			return
		}
	}

	tx.Commit()
	ctx.JSON(200, gin.H{"status": "Красава, всё загружено"})
}
