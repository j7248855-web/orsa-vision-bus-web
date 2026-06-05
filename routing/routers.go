package routers

import (
	"orsavisionweb/internal/auth"
	"orsavisionweb/internal/core/reports"
	"orsavisionweb/internal/middleware"
	"orsavisionweb/internal/modules/bus"
	"orsavisionweb/internal/modules/devices"
	"orsavisionweb/internal/modules/routes"
	"orsavisionweb/internal/modules/stops"
	"orsavisionweb/internal/modules/users"

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
	//------------------------- REST для пользователя  --------------------------
	//Добавление нового пользователя
	pr.POST("/new/user", func(ctx *gin.Context) {
		users.CreateNewUser(ctx, conn)
	})
	//Возвращение нового пользователя
	pr.GET("/new/user", func(ctx *gin.Context) {
		users.ReturnNewUser(ctx, conn)
	})
	//Удаление пользователя
	pr.DELETE("/remove/user/:user_id", func(ctx *gin.Context) {
		users.RemoveUser(ctx, conn)
	})
	//Редактирование пользователей
	pr.PUT("/edit/user", func(ctx *gin.Context) {
		users.EditUser(ctx, conn)
	})
	//------------------------------------------------------------------------------

	//------------------------------- REST ДЛЯ МАРШРУТОВ ---------------------------
	//Загрузка данных об маршрутах
	pr.POST("/routes", func(ctx *gin.Context) {
		routes.HandleRouteWithPoints(ctx, conn)
	})
	//Получение всех маршрутов
	pr.GET("/data/routes", func(ctx *gin.Context) {
		routes.FullTripsData(ctx, conn)
	})
	//Получение определённого маршрта для редактирования
	pr.GET("/routes/:route_id", func(ctx *gin.Context) {
		routes.GetTripInformation(ctx, conn)
	})
	//Удаление маршрутов
	pr.DELETE("/remove/routes/:route_id", func(ctx *gin.Context) {
		routes.RemoveRoutes(ctx, conn)
	})
	pr.PUT("/edit/routes", func(ctx *gin.Context) {
		routes.EditRoutes(ctx, conn)
	})
	//------------------------------------------------------------------------------

	//------------------------------- REST ДЛЯ ОСТАНОВОК ---------------------------
	//Добавление новых остановок
	pr.POST("/routes/stops", func(ctx *gin.Context) {
		stops.HandleRouteStops(ctx, conn)
	})
	//Редактирование остановок
	pr.PUT("/edit/stops", func(ctx *gin.Context) {
		stops.EditBusStops(ctx, conn)
	})
	//Возврат данных об остановках по определённому городу
	pr.GET("/stops/:city", func(ctx *gin.Context) {
		stops.FullBusStation(ctx, conn)
	})
	pr.DELETE("/stops/remove/:stop_id", func(ctx *gin.Context) {
		stops.RemoveStops(ctx, conn)
	})
	//------------------------------------------------------------------------------

	//------------------------------- REST ДЛЯ АВТОБУСОВ ---------------------------

	//Регистрация нового автобуса и его девайсов
	pr.POST("/new/bus", func(ctx *gin.Context) {
		bus.RegisterBus(ctx, conn)
	})
	//Перечень доступных автобусов и их девайсов
	pr.GET("/new/bus", func(ctx *gin.Context) {
		bus.GetBuses(ctx, conn)
	})
	pr.PUT("/edit/bus", func(ctx *gin.Context) {
		bus.EditBus(ctx, conn)
	})
	//Удаление автобуса
	pr.DELETE("/remove/bus/:bus_id", func(ctx *gin.Context) {
		bus.RemoveBus(ctx, conn)
	})
	pr.GET("/data/bus/:route_number", func(ctx *gin.Context) {
		bus.DataBus(ctx, conn)
	})

	//------------------------------------------------------------------------------------

	//------------------------------- REST ДЛЯ ДЕВАЙСОВ ----------------------------------
	//Обновление девайсов
	pr.PUT("/edit/device/:device_id", func(ctx *gin.Context) {
		devices.DeviceEditor(ctx, conn)
	})
	//Удаление девайсов
	pr.DELETE("/delete/device/:device_id", func(ctx *gin.Context) {
		devices.DeviceRemove(ctx, conn)
	})
	//-------------------------------------------------------------------------------------

	//---------------------------- REST ДЛЯ ОТПРАВКИ ОТЧЁТА -------------------------------
	//Отправка файла отчёта
	pr.POST("/journey/reports/download", func(ctx *gin.Context) {
		reports.HandleGenerateReport(ctx, conn)
	})
	//Отправка всей информации об отчёте
	pr.POST("/journey/reports/list", func(ctx *gin.Context) {
		reports.HandleGetReportsList(ctx, conn)
	})
	//-------------------------------------------------------------------------------------

	//---------------------------REST ДЛЯ ТРЕВОЖНОЙ КНОПКИ---------------------------------
	pr.GET("/emergency/data", func(ctx *gin.Context) {
		bus.EmergencyButton(ctx, conn)
	})
}
