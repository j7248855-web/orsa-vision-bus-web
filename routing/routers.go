package routers

import (
	"orsavisionweb/internal/auth"
	"orsavisionweb/internal/handler"
	"orsavisionweb/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func Routing(r *gin.Engine, conn *sqlx.DB) {
	//Логика входа в аккаунт
	r.POST("/auth", func(ctx *gin.Context) {
		auth.Login(ctx, conn)
	})
	//Логика защиты
	protected := r.Group("/api")
	pr := protected.Use(middleware.MiddleWareAuth)
	pr.POST("/new/user", func(ctx *gin.Context) {
		handler.CreateNewUser(ctx, conn)
	})
	pr.GET("/new/user", func(ctx *gin.Context) {
		handler.ReturnNewUser(ctx, conn)
	})
	//Загрузка данных об маршрутах
	pr.POST("/routes", func(ctx *gin.Context) {
		handler.HandleRouteWithPoints(ctx, conn)
	})
	pr.GET("/routes", func(ctx *gin.Context) {
		handler.GetFullRoutes(ctx, conn)
	})
	//Добавление новых остановок
	pr.POST("/routes/stops", func(ctx *gin.Context) {
		handler.HandleRouteStops(ctx, conn)
	})
	//Возврат данных об остановках по определённому городу
	pr.GET("/stops/:city", func(ctx *gin.Context) {
		handler.FullBusStation(ctx, conn)
	})
	//Регистрация нового автобуса и его девайсов
	pr.POST("/new/bus", func(ctx *gin.Context) {
		handler.RegisterBus(ctx, conn)
	})
	//Перечень доступных автобусов и их девайсов
	pr.GET("/new/bus", func(ctx *gin.Context) {
		handler.GetBuses(ctx, conn)
	})
}
