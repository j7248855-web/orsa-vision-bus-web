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
	StopStates   map[string]*models.Dependence
}

func (serv *GPSServer) Stream(cx context.Context, req *gps_pt.GPSData) (*gps_pt.Status, error) {
	serv.Mu.Lock()

	if serv.Storage == nil {
		serv.Storage = make(map[string]*models.BusContext)
	}
	if serv.NonExisenIPs == nil {
		serv.NonExisenIPs = make(map[string]time.Time)
	}
	if serv.StopStates == nil {
		serv.StopStates = make(map[string]*models.Dependence)
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

		state.LastPoint = currentPoint
		state.LastTime = actualTime

		query := `
            SELECT b.bus_number, d.device_ip
            FROM routes r
            JOIN buses b ON b.route_number = r.route_number
            JOIN devices d ON d.bus_id = b.id
            WHERE b.route_number = $1 AND d.type = 'teltonic' AND d.status = 'active' AND b.status = 'active'`

		var dbRows []struct {
			BusNumber string `db:"bus_number"`
			DeviceIP  string `db:"device_ip"`
		}

		if err := serv.DB.Select(&dbRows, query, busCtx.RouteNumber); err != nil {
			log.Println("Ошибка получения автобусов маршрута:", err)
		} else {

			busesMap := make([]models.BusOnMap, 0, len(dbRows))

			serv.Mu.Lock()
			for _, row := range dbRows {
				ctxInStorage, ok := serv.Storage[row.DeviceIP]
				var busLat, busLng float64

				if ok && ctxInStorage.State != nil && ctxInStorage.State.LastPoint != nil {
					busLat = ctxInStorage.State.LastPoint[0]
					busLng = ctxInStorage.State.LastPoint[1]
				}

				busesMap = append(busesMap, models.BusOnMap{
					BusNumber: row.BusNumber,
					Lat:       busLat,
					Lng:       busLng,
				})
			}
			serv.Mu.Unlock()

			go serv.Conns.SendLocation(busesMap)
		}

		if busCtx.State.TripStatus == "in_trip" {
			logic.ProcessRouteDeviation(serv.DB, busCtx, currentPoint, actualTime)
		}

		var timeDiff time.Duration
		timeDiff = actualTime.Sub(state.LastTime)

		if busCtx.State.TripStatus == "in_trip" {
			for _, v := range busCtx.Stop {
				stopPos := []float64{v.Lat, v.Lon}
				stateKey := fmt.Sprintf("%d_%d", busCtx.BusID, v.ID)

				serv.Mu.Lock()
				stopState, exists := serv.StopStates[stateKey]
				if !exists {
					stopState = &models.Dependence{}
					serv.StopStates[stateKey] = stopState
				}
				serv.Mu.Unlock()

				event := logic.CalculateStopStation(stopState, currentPoint, state.LastPoint, timeDiff, stopPos, v.Radius, actualTime, busCourse, v.Azimuth)
				if event != nil {
					logic.LogStopEvent(serv.DB, busCtx, v, event)

					if event.IsSkipped {
						reports.ViolationsReport(serv.DB, busCtx, "Пропуск остановки", fmt.Sprintf("Остановка \"%v\" пропущена", v.Name))
					}

					serv.Mu.Lock()
					delete(serv.StopStates, stateKey)
					serv.Mu.Unlock()
				}
			}
		}
	case *gps_pt.GPSData_Gga:
		log.Println("Пришли GGA:", data.Gga)
	case nil:
		log.Println("Ожидание данных")
	}
	return &gps_pt.Status{Status: true}, nil
}
