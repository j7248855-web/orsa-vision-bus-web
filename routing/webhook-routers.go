package routers

import (
	"log"
	core "orsavisionweb/internal/core/wh"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Функция вебхука тревожной кнопки
func WHRoute(serv *gin.Engine, conn *sqlx.DB) {
	serv.GET("/api/alarm", func(ctx *gin.Context) {
		log.Println("Запрос от тревожной кнопки")
		rawData, err := ctx.GetRawData()
		if err == nil && len(rawData) > 0 {
			log.Println("ДАННЫЕ ОТ ЗАПРОСА:", string(rawData))
			core.BusEmergency(rawData, conn)
		}
		ctx.JSON(200, gin.H{
			"status": "success",
		})
	})
}
