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
	Storage map[string]*models.BusContext
	DB      *sqlx.DB
	Conns   *ws.Broadcaster
	Mu      sync.Mutex
}

func (serv *GPSServer) Stream(cx context.Context, req *gps_pt.GPSData) (*gps_pt.Status, error) {
	if serv.Storage == nil {
		serv.Storage = make(map[string]*models.BusContext)
	}
	log.Printf("IP ОТПРАВИТЕЛЯ (req.DeviceIp): '%s'", req.DeviceIp)
	//Берём данные про айпишник
	serv.Mu.Lock()
	busCtx, ok := serv.Storage[req.DeviceIp]
	serv.Mu.Unlock()
	if !ok {
		busCtx = auxuliary.LoadFullBusData(req.DeviceIp, serv.DB)
		if busCtx == nil {
			return &gps_pt.Status{Status: false}, nil
		}
		serv.Mu.Lock()
		serv.Storage[req.DeviceIp] = busCtx
		serv.Mu.Unlock()
	}

	state := busCtx.State
	//Взять busID от базы (IP)
	switch data := req.Payload.(type) {
	//Определяем что за херня чтобы отправить дальше
	case *gps_pt.GPSData_Rmc:
		lat := auxuliary.ConvertNMEAToDecimal(data.Rmc.Lat, data.Rmc.LatDir)
		lon := auxuliary.ConvertNMEAToDecimal(data.Rmc.Lon, data.Rmc.LonDir)

		currentPoint := []float64{lat, lon}
		if state.LastPoint == nil {
			state.LastPoint = currentPoint
		}
		//Триггер на просмотр остановки автобуса, на автобусных остановках
		now := time.Now()
		gpsTime, err := time.Parse("150405.00", data.Rmc.Utc)
		if err != nil {
			log.Println("Не удалось распарсить дату")
		}
		actualTime := time.Date(
			now.Year(), now.Month(), now.Day(),
			gpsTime.Hour(), gpsTime.Minute(), gpsTime.Second(), 0, time.UTC)
		//Вычисление отклонения маршрута от нормы
		deviation := logic.CheckDeviation(lat, lon, busCtx.Points)
		var timeDiff time.Duration
		if state.LastTime.IsZero() {
			state.LastTime = actualTime
		}
		timeDiff = actualTime.Sub(state.LastTime)
		//Берём азимут остановки
		busCourse, _ := strconv.ParseFloat(data.Rmc.TrackTrue, 64)

		//Рассчёт вхождения
		logic.ProcessTripState(serv.DB, busCtx, currentPoint, actualTime)
		go serv.Conns.SendLocation(gin.H{
			"bus_id": busCtx.BusID,
			"lat":    lat,
			"lon":    lon,
			"course": busCourse,
		})
		if deviation.IsOffRoute {
			reports.ViolationsReport(serv.DB, busCtx, "Выход с маршрута", deviation.Value)
		}
		for _, v := range busCtx.Stop {
			stopPos := []float64{v.Lat, v.Lon}
			wasAtStop := state.IsBusStop //Смотрим ли был он на этой остановке до расчёта
			event := logic.CalculateStopStation(state, currentPoint, state.LastPoint, timeDiff, stopPos, v.Radius, actualTime, busCourse, v.Azimuth)
			//Вычисление времени прибытия автобуса на остановку
			//Формируем отчёт по остановкам
			if event != nil {
				logic.LogStopEvent(serv.DB, busCtx, v, event)

				if event.IsSkipped { //Отправка нарушения по остановкам
					reports.ViolationsReport(serv.DB, busCtx, "Пропуск остановки ", fmt.Sprintf("Остановка \"%v\" пропущена", v.Name))
				}

				if !wasAtStop && state.IsBusStop && !event.IsSkipped {
					delay := logic.CalculateDelay(event, v.Schedule)
					if delay > 5 { // Порог 5 минут
						reports.ViolationsReport(serv.DB, busCtx, "Нарушение графика", fmt.Sprintf("+%d мин", delay))
					}
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
