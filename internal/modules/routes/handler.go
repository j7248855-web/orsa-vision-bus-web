package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"orsavisionweb/internal/models"
	auxuliary "orsavisionweb/internal/utils/auxiliary"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Функция загрузки данных о маршруте
// Загружаются данныые по точкам маршрута, и точкам остановок
func HandleRouteWithPoints(ctx *gin.Context, conn *sqlx.DB) {
	//Достаём от FormData данные по структуре
	routeJsonStr := ctx.PostForm("route")
	if routeJsonStr == "" {
		log.Println("Отсутствует ключ route")
		ctx.JSON(400, gin.H{"status:": "Отсутствует ключ route"})
		return
	}
	var route models.Route

	err := json.Unmarshal([]byte(routeJsonStr), &route)
	fmt.Println("Данные от маршрута:", route)
	if err != nil {
		log.Printf("Не удалось получить данные от JSON: %v", err)
		ctx.JSON(400, gin.H{"status:": "Неправильный JSON внутри FormData", "details": err.Error()})
		return
	}

	tx, err := conn.Beginx()
	if err != nil {
		log.Println("Ошибка транзакции", err)
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		return
	}
	defer tx.Rollback()

	var actualRouteID int
	err = tx.QueryRowContext(ctx, `
        INSERT INTO routes (route_number, name, city, status) 
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (route_number, city) DO UPDATE SET 
            name = EXCLUDED.name, 
            status = EXCLUDED.status
        RETURNING id`,
		route.RouteID, route.Name, route.City, route.Status).Scan(&actualRouteID)

	if err != nil {
		ctx.JSON(500, gin.H{"error": "Проблема добавления" + err.Error()})
		return
	}
	//Быстаря очистка старых точек маршрута, и замену на новые
	_, err = tx.ExecContext(ctx, "DELETE FROM route_path_points WHERE route_id = $1", actualRouteID)
	if err != nil {
		ctx.JSON(500, gin.H{"status:": "Не удалось очистить точки маршрута", "details": err.Error()})
		return
	}
	//Быстрая вставка в таблицу за единственный проход
	var tempArrForPoints = make([]models.RoutePoint, 0, len(route.Points))
	for i, p := range route.Points {
		tempArrForPoints = append(tempArrForPoints, models.RoutePoint{
			RouteID:       actualRouteID,
			Lat:           p[1],
			Lon:           p[0],
			SequenceOrder: i,
		})
	}
	if len(tempArrForPoints) > 0 {
		_, err := tx.NamedExecContext(ctx, "INSERT INTO route_path_points (route_id, lat, lng, sequence_order) VALUES (:route_id, :lat, :lng, :sequence_order)", tempArrForPoints)
		if err != nil {
			log.Println("Проблема с добавлением в базу:", err)
			ctx.JSON(500, gin.H{"status:": "Не удалось добавить данные в базу", "details": err.Error()})
		}
	}
	schedule := auxuliary.ParsingScheduleCSV(ctx)
	if len(schedule.Trips) == 0 {
		return
	}
	var flatSchedules []models.TripSteps
	for _, currentGraph := range schedule.Trips {
		for _, step := range currentGraph {
			step.RouteID = actualRouteID

			flatSchedules = append(flatSchedules, step)
		}
	}
	if len(flatSchedules) > 0 {
		_, err = tx.NamedExecContext(ctx, `INSERT INTO schedules (route_id, sequence_number, departure_time, arrival_time) 
    VALUES (:route_id, :sequence_number, :departure_time, :arrival_time)`, flatSchedules)
		//Быстаря очистка старых точек остановок, и замену на новые
		_, err = tx.ExecContext(ctx, "DELETE FROM route_stops WHERE route_id = $1", actualRouteID)
		if err != nil {
			ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
			return
		}
	}

	for i, stop := range route.Stops {
		_, err = tx.ExecContext(ctx, `
            INSERT INTO route_stops (route_id, stop_id, sequence_order, type) 
            VALUES ($1, $2, $3, $4)`,
			actualRouteID, stop.ID, i, stop.Type)

		if err != nil {
			log.Println("Не удалось добавить остановки для маршрута:", err)
			ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Непредвиденная ошибка: %v", err)
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		return
	}
	schedule.RouteID = actualRouteID
	ctx.JSON(200, gin.H{"status": "success", "schedules": schedule})
}

// Отправка всех данных по маршруту
func GetTripInformation(ctx *gin.Context, conn *sqlx.DB) {
	var routes models.Route
	var routeID = ctx.Param("route_id")
	var points []models.RoutePoint
	var stops []models.Stop

	//Взятие данных об маршруте
	err := conn.GetContext(ctx, &routes, "SELECT id, route_number, name, city, status FROM routes WHERE id=$1", routeID)
	if err != nil {
		log.Println("Непредвиденная ошибка при получении маршрутов:", err)
		ctx.JSON(500, gin.H{"status": "Непредвиденная ошибка", "details": err.Error()})
		return
	}

	//Взятие всех точек маршрта
	err = conn.SelectContext(ctx, &points, "SELECT lat, lng FROM route_path_points WHERE route_id=$1 ORDER BY sequence_order", routeID)
	if err != nil {
		log.Println("Непредвиденная ошибка при получении маршрутов:", err)
		ctx.JSON(500, gin.H{"status": "Непредвиденная ошибка", "details": err.Error()})
		return
	}

	//Взятие всех данных об остановках
	err = conn.SelectContext(ctx, &stops, `SELECT 
				s.id, 
				s.name, 
				s.lat, 
				s.lng, 
				s.radius, 
				rs.type, 
				s.azimuth, 
				s.city 
			FROM stops s
			JOIN route_stops rs ON s.id = rs.stop_id
			WHERE rs.route_id = $1
			ORDER BY rs.sequence_order`, routeID)
	if err != nil {
		log.Println("Непредвиденная ошибка при получении остановок:", err)
		ctx.JSON(500, gin.H{"status": "Непредвиденная ошибка", "details": err.Error()})
		return
	}
	routes.Stops = make([]models.Stop, 0, len(stops))
	for _, v := range stops {
		v.Position = [2]float64{v.Lon, v.Lat}
		routes.Stops = append(routes.Stops, v)
	}
	routes.Points = make([][2]float64, 0, len(points))
	for _, v := range points {
		pair := [2]float64{v.Lon, v.Lat}
		routes.Points = append(routes.Points, pair)
	}
	ctx.JSON(200, routes)
}

// Полное редактирование данных об маршруте
func EditRoutes(ctx *gin.Context, conn *sqlx.DB) {
	var routes models.Route

	if err := ctx.ShouldBindJSON(&routes); err != nil {
		log.Println("Ошибка парсинга JSON:", err)
		ctx.JSON(400, gin.H{"error": "Не удалось распарсить JSON", "details": err.Error()})
		return
	}

	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		log.Println("Ошибка старта транзакции:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка базы данных"})
		return
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `UPDATE routes SET route_number = $1, name = $2, city = $3, status = $4 WHERE id = $5`, routes.RouteID, routes.Name, routes.City, routes.Status, routes.ID)
	if err != nil {
		log.Println("Непредвиденная ошибка обновлении маршрута:", err)
		ctx.JSON(500, gin.H{"status": "Непредвиденная ошибка", "details": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		ctx.JSON(404, gin.H{"error": "Маршрут с таким ID не найден"})
		return
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM route_path_points WHERE route_id = $1", routes.ID)
	if err != nil {
		log.Println("Непредвиденная ошибка удалении старых точек:", err)
		ctx.JSON(500, gin.H{"status": "Непредвиденная ошибка", "details": err.Error()})
		return
	}

	var tempArrForPoints = make([]models.RoutePoint, 0, len(routes.Points))
	for i, p := range routes.Points {
		tempArrForPoints = append(tempArrForPoints, models.RoutePoint{
			RouteID:       routes.ID,
			Lat:           p[1],
			Lon:           p[0],
			SequenceOrder: i,
		})
	}

	if len(tempArrForPoints) > 0 {
		insertPointQuery := `
            INSERT INTO route_path_points (route_id, lat, lng, sequence_order) 
            VALUES (:route_id, :lat, :lng, :sequence_order)`

		for _, point := range tempArrForPoints {
			_, err := tx.NamedExecContext(ctx, insertPointQuery, point)
			if err != nil {
				log.Println("Проблема с добавлением в базу через NamedExec:", err)
				ctx.JSON(500, gin.H{"status": "Не удалось добавить данные в базу", "details": err.Error()})
				return
			}
		}
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM route_stops WHERE route_id = $1", routes.ID)
	if err != nil {
		log.Println("Ошибка удаления старых связей остановок:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка обновления остановок: " + err.Error()})
		return
	}

	for i, stop := range routes.Stops {
		lat := stop.Position[1]
		lng := stop.Position[0]

		if lat == 0 && lng == 0 {
			lat = stop.Lat
			lng = stop.Lon
		}

		var stopID int
		if stop.ID != 0 {
			stopID = stop.ID
			_, err = tx.ExecContext(ctx, "UPDATE stops SET lat = $1, lng = $2, name = $3, type = $4, radius = $5 WHERE id = $6", lat, lng, stop.Name, stop.Type, stop.Radius, stopID)
			if err != nil {
				log.Println("Ошибка обновления параметров остановки:", err)
				ctx.JSON(500, gin.H{"error": "Ошибка обновления остановки: " + err.Error()})
				return
			}
		} else {
			queryInsertStop := `
				INSERT INTO stops (name, type, lat, lng, radius) 
				VALUES ($1, $2, $3, $4, $5) RETURNING id`
			err = tx.QueryRowContext(ctx, queryInsertStop, stop.Name, stop.Type, lat, lng, stop.Radius).Scan(&stopID)
			if err != nil {
				log.Println("Ошибка создания новой остановки:", err)
				ctx.JSON(500, gin.H{"error": "Ошибка создания остановки: " + err.Error()})
				return
			}
		}

		_, err = tx.ExecContext(ctx, "INSERT INTO route_stops (route_id, stop_id, sequence_order) VALUES ($1, $2, $3)", routes.ID, stopID, i)
		if err != nil {
			log.Println("Ошибка связывания остановки с маршрутом:", err)
			ctx.JSON(500, gin.H{"error": "Ошибка сохранения структуры остановок: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Println("Проблема закрытия транзакциии:", err)
		ctx.JSON(500, gin.H{"error": "Не удалось зафиксировать данные в БД: " + err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"status": "success"})
}

// Отдача всех данных по маршруту
func FullTripsData(ctx *gin.Context, conn *sqlx.DB) {
	var routes []models.Route
	//Взятие данных об маршруте
	err := conn.SelectContext(ctx, &routes, "SELECT id, route_number, name, city, status FROM routes")
	if err != nil {
		log.Println("Непредвиденная ошибка при получении маршрутов:", err)
		ctx.JSON(500, gin.H{"status": "Непредвиденная ошибка", "details": err.Error()})
		return
	}
	ctx.JSON(200, routes)
}

// Удаление точек маршрута
func RemoveRoutes(ctx *gin.Context, conn *sqlx.DB) {
	routeID := ctx.Param("route_id")

	query := `DELETE FROM routes WHERE id = $1`
	result, err := conn.ExecContext(ctx, query, routeID)
	if err != nil {
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		ctx.JSON(404, gin.H{"status:": "Непредвиденная ошибка"})
		return
	}

	ctx.JSON(200, gin.H{"status": "success"})
}
