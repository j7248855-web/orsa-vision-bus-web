package routes

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Функция загрузки данных о маршруте
// Загружаются данныые по точкам маршрута, и точкам остановок
func HandleRouteWithPoints(ctx *gin.Context, conn *sqlx.DB) {
	var route models.Route

	if err := ctx.ShouldBindJSON(&route); err != nil {
		log.Printf("Непредвиденная ошибка: %v", err)
		ctx.JSON(400, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		return
	}

	tx, err := conn.Beginx()
	if err != nil {
		log.Printf("Ошибка транзакции: %v", err)
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
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
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
		tx.NamedExecContext(ctx, "INSERT INTO route_path_points (route_id, lat, lng, sequence_order) VALUES (:route_id, :lat, :lng, :sequence_order)", tempArrForPoints)
		if err != nil {
			log.Println("Проблема с добавлением в базу:", err)
			ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		}
	}
	//Быстаря очистка старых точек остановок, и замену на новые
	_, err = tx.ExecContext(ctx, "DELETE FROM route_stops WHERE route_id = $1", actualRouteID)
	if err != nil {
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		return
	}
	for i, stop := range route.Stops {
		_, err = tx.ExecContext(ctx, `
            INSERT INTO route_stops (route_id, stop_id, sequence_order) 
            VALUES ($1, $2, $3)`,
			actualRouteID, stop.ID, i)

		if err != nil {
			log.Printf("Не удалось добавить остановки для маршрута:", err)
			ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Непредвиденная ошибка: %v", err)
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"status": "success", "id": actualRouteID})
}

// Отправка всех данных по маршруту
func GetFullRoutes(ctx *gin.Context, conn *sqlx.DB) {
	routeID := ctx.Param("route_id")
	var routes models.Route

	err := conn.GetContext(ctx, &routes, "SELECT id, route_number, name, status FROM routes WHERE id=$1", routeID)
	if err != nil {
		log.Printf("Непредвиденная ошибка: %v", err)
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		return
	}
	//Временна структура для хранения всех точек
	var points []struct {
		Lat float64 `db:"lat"`
		Lng float64 `db:"lng"`
	}

	//Берём из базы все точки маршрута
	err = conn.SelectContext(ctx, &points,
		"SELECT lat, lng FROM route_path_points WHERE route_id = $1 ORDER BY sequence_order", routeID)
	if err != nil {
		log.Println("Не удалось достать данные по маршруту, id маршурат:", routeID)
		ctx.JSON(500, gin.H{"status:": "Непредвиденная ошибка", "details": err.Error()})
		return
	} else {
		routes.Points = make([][2]float64, 0, len(points))
		for _, p := range points {
			routes.Points = append(routes.Points, [2]float64{p.Lng, p.Lat})
		}
	}
	err = conn.SelectContext(ctx, &routes.Stops, `
        SELECT s.id, s.name, s.lat, s.lng, s.radius, s.type, s.azimuth, s.city 
        FROM stops s
        JOIN route_stops rs ON s.id = rs.stop_id
        WHERE rs.route_id = $1
        ORDER BY rs.sequence_order`, routeID)
	if err != nil {
		log.Printf("Ошибка поиска остановок: %v", err)
		routes.Stops = []models.Stop{}
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
