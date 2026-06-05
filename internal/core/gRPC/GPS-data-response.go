package handler

import (
	"context"
	"fmt"
	"log"
	"orsavisionweb/internal/core/logic"
	"orsavisionweb/internal/core/reports"
	"orsavisionweb/internal/core/ws"
	"orsavisionweb/internal/models"
	auxuliary "orsavisionweb/internal/utils/auxiliary"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	gps_pt "github.com/j7248855-web/orsa-vision-grpc-second/gen/sso"
	"github.com/jmoiron/sqlx"
)

type GPSServer struct {
	gps_pt.UnimplementedGPSTrackerServer
	Storage      map[string]*models.BusContext
	NonExisenIPs map[string]time.Time
	DB           *sqlx.DB
	Conns        *ws.Broadcaster
	Mu           sync.Mutex
}

func (serv *GPSServer) Stream(cx context.Context, req *gps_pt.GPSData) (*gps_pt.Status, error) {
	serv.Mu.Lock()
	log.Printf("Получен пакет от IP: %s, тип Payload: %T", req.DeviceIp, req.Payload)

	if serv.Storage == nil {
		serv.Storage = make(map[string]*models.BusContext)
	}
	if serv.NonExisenIPs == nil {
		serv.NonExisenIPs = make(map[string]time.Time)
	}
	blockedUntil, isBlocked := serv.NonExisenIPs[req.DeviceIp]

	if isBlocked {
		if time.Now().Before(blockedUntil) {
			serv.Mu.Unlock()
			return &gps_pt.Status{Status: false}, nil
		}
		delete(serv.NonExisenIPs, req.DeviceIp)
	}

	busCtx, ok := serv.Storage[req.DeviceIp]
	if !ok {
		busCtx = auxuliary.LoadFullBusData(req.DeviceIp, serv.DB)
		if busCtx == nil {
			serv.NonExisenIPs[req.DeviceIp] = time.Now().Add(10 * time.Minute)
			serv.Mu.Unlock()
			return &gps_pt.Status{Status: false}, nil
		}
		serv.Storage[req.DeviceIp] = busCtx
	}
	serv.Mu.Unlock()

	state := busCtx.State

	switch data := req.Payload.(type) {
	case *gps_pt.GPSData_Rmc:
		busCourse, _ := strconv.ParseFloat(data.Rmc.TrackTrue, 64)
		lat := auxuliary.ConvertNMEAToDecimal(data.Rmc.Lat, data.Rmc.LatDir)
		lon := auxuliary.ConvertNMEAToDecimal(data.Rmc.Lon, data.Rmc.LonDir)

		currentPoint := []float64{lat, lon}
		if state.LastPoint == nil {
			state.LastPoint = currentPoint
		}

		now := time.Now()
		gpsTime, err := time.Parse("150405.00", data.Rmc.Utc)
		if err != nil {
			log.Println("Не удалось распарсить дату")
		}
		actualTime := time.Date(
			now.Year(), now.Month(), now.Day(),
			gpsTime.Hour(), gpsTime.Minute(), gpsTime.Second(), 0, time.UTC)

		logic.ProcessTripState(serv.DB, busCtx, currentPoint, actualTime)

		log.Printf("[DEBUG] Автобус ID: %d, Статус рейса: '%s', Текущая конечная ID: %d", busCtx.BusID, state.TripStatus, state.CurrentStartStopID)

		go serv.Conns.SendLocation(gin.H{
			"bus_id": busCtx.BusID,
			"lat":    lat,
			"lon":    lon,
			"course": busCourse,
		})

		var deviation models.DeviationResult
		if busCtx.State.TripStatus == "in_trip" {
			deviation = logic.CheckDeviation(lat, lon, busCtx.Points)
		}
		var timeDiff time.Duration
		if state.LastTime.IsZero() {
			state.LastTime = actualTime
		}
		timeDiff = actualTime.Sub(state.LastTime)

		if busCtx.State.TripStatus == "in_trip" {
			if deviation.IsOffRoute {
				log.Printf("Фиксация выхода с маршрута для автобуса %d, Значение: %v", busCtx.BusID, deviation.Value)
				reports.ViolationsReport(serv.DB, busCtx, "Выход с маршрута", deviation.Value)
			}

			// Чтобы не падать из-за отсутствия структур, используем логику "в моменте"
			for _, v := range busCtx.Stop {
				stopPos := []float64{v.Lat, v.Lon}

				// Нам важно знать, был ли автобус на ЭТОЙ остановке.
				// Твой старый код использовал общий state.IsBusStop, который ломался в цикле.
				// Временно подменяем вызов, передавая v.ID, чтобы логика внутри понимала, с какой остановкой работает.
				var event *models.StopEvent
				if timeDiff > 0 {
					event = logic.CalculateStopStation(state, currentPoint, state.LastPoint, timeDiff, stopPos, v.Radius, actualTime, busCourse, v.Azimuth, v.ID)
				}

				if event != nil {
					// Симулируем, что автобус остановился, чтобы затриггерить генерацию твоих отчетов
					state.IsBusStop = true

					logic.LogStopEvent(serv.DB, busCtx, v, event)

					if event.IsSkipped {
						log.Printf("[REPORTS] Фиксация пропуска остановки: %s", v.Name)
						reports.ViolationsReport(serv.DB, busCtx, "Пропуск остановки ", fmt.Sprintf("Остановка \"%v\" пропущена", v.Name))
					}

					// Твой старый блок генерации нарушения графика
					delay := logic.CalculateDelay(event, v.Schedule)
					if delay > 5 {
						log.Printf("[REPORTS] Фиксация нарушения графика: +%d мин", delay)
						reports.ViolationsReport(serv.DB, busCtx, "Нарушение графика", fmt.Sprintf("+%d мин", delay))
					}

					state.IsBusStop = false // Сбрасываем обратно
				}
			}
		}
		state.LastPoint = currentPoint
		state.LastTime = actualTime
	case *gps_pt.GPSData_Gga:
		log.Println("Пришли GGA:", data.Gga)
	case nil:
		log.Println("Ожидание данных")
	}
	return &gps_pt.Status{Status: true}, nil
}
