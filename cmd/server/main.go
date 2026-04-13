package main

import (
	"fmt"
	"log"
	"net"
	"orsavisionweb/internal/database"
	"orsavisionweb/internal/handler"
	routers "orsavisionweb/routing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	gps_pt "github.com/j7248855-web/orsa-vision-grpc-second/gen/sso"
	cam_pt "github.com/j7248855-web/orsa-vision-grpc-third/gen/cam"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	err := godotenv.Load("../../configs/database.env")
	if err != nil {
		log.Println("Не удалось прочитать .ENV файл", err)
		return
	}
	dbConn := database.Connection()
	defer dbConn.Close()
	//Чтение GPS
	go func() {
		lis, _ := net.Listen("tcp", ":8585")
		fmt.Println("TCP соединение создано, ожидаю GPS")
		servGRPC := grpc.NewServer()
		gps_pt.RegisterGPSTrackerServer(servGRPC, &handler.GPSServer{})
		cam_pt.RegisterCameraControlServer(servGRPC, &handler.Server{DB: dbConn})
		servGRPC.Serve(lis)
	}()
	//Инициализация gin сервера
	serv := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true // Для разработки пойдет
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	serv.Use(cors.New(config))
	//Подключение роутеров
	routers.Routing(serv, dbConn)
	serv.Run(":8686")
}
