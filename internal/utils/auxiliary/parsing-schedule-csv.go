package auxuliary

import (
	"encoding/csv"
	"io"
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
)

func ParsingScheduleCSV(ctx *gin.Context) models.Schedule {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		log.Println("Не удалось получить файл CSV:", err)
		ctx.JSON(400, gin.H{"status": "error", "details": err.Error()})
		return models.Schedule{}
	}
	file, err := fileHeader.Open()
	if err != nil {
		log.Println("Не удалось открыть файл CSV:", err)
		ctx.JSON(400, gin.H{"status": "error", "details": err.Error()})
		return models.Schedule{}
	}
	//Читаем сам файл
	var currentSequence = 1
	var schedule models.Schedule
	var tripSchedule models.TripSteps
	var arrTripSchedule []models.TripSteps
	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Println("Не удалось прочитать график из файла:", err)
			return models.Schedule{}
		}
		if record[0] == "" || len(record[0]) == 0 {
			tripSchedule.SequenceNumber = currentSequence
			currentSequence++
			schedule.Trips = append(schedule.Trips, arrTripSchedule)
			arrTripSchedule = []models.TripSteps{}
			continue
		}
		//записываем данные от графика в матрицу
		tripSchedule.DepartureTime = record[0]
		tripSchedule.ArrivalTime = record[1]
		arrTripSchedule = append(arrTripSchedule, tripSchedule)

	}

	if len(arrTripSchedule) > 0 {
		schedule.Trips = append(schedule.Trips, arrTripSchedule)
	}
	return schedule
}
