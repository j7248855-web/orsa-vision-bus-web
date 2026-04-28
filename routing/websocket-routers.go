package routers

import (
	"log"
	"net/http"
	"orsavisionweb/internal/core/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func WSRoute(r *gin.Engine, b *ws.Broadcaster) {
	r.GET("/ws/bus", func(ctx *gin.Context) {
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

		//Держим соединение пока сокет живой
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	})
}
