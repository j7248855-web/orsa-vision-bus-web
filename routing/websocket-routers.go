package routers

import (
	"log"
	"net/http"
	"orsavisionweb/internal/core/ws"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
)

func WSRoute(r *gin.Engine, b *ws.Broadcaster, db *sqlx.DB) {
	r.GET("/ws/bus", func(ctx *gin.Context) {
		routeID := ctx.Query("route_id")
		if routeID == "" {
			log.Println("route_id не указан")
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "route_id обязателен"})
			return
		}

		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			return
		}

		// Регистрируем клиента
		b.Mu.Lock()
		b.Clients[conn] = true
		b.Mu.Unlock()

		// Чистим за собой при выходе из функции
		defer func() {
			b.Mu.Lock()
			delete(b.Clients, conn)
			b.Mu.Unlock()
			conn.Close()
			log.Println("Клиент отключился от WS")
		}()

		// Достаем автобусы и их телтоники по цепочке
		query := `
			SELECT b.bus_number
			FROM routes r
			JOIN buses b ON b.route_number = r.route_number
			JOIN devices d ON d.bus_id = b.id
			WHERE r.id = $1 AND d.type = 'teltonic' AND d.status = 'active' AND b.status = 'active'`

		var dbRows []struct {
			BusNumber string `db:"bus_number"`
		}

		if err := db.SelectContext(ctx, &dbRows, query, routeID); err != nil {
			log.Println("Ошибка при выборке автобусов для WS:", err)
		} else {
			// Перекладываем в нужный фронту массив структур
			buses := make([]models.BusOnMap, 0, len(dbRows))
			for _, row := range dbRows {
				buses = append(buses, models.BusOnMap{
					BusNumber: row.BusNumber,
					Lat:       0.0, // Дефолтные координаты, далее они обновятся через SendLocation
					Lng:       0.0,
				})
			}

			// Отправляем сформированный массив первично в сокет
			if err := conn.WriteJSON(buses); err != nil {
				log.Println("Ошибка отправки данных в сокет:", err)
			}
		}

		// Держим соединение пока сокет живой
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	})
}
