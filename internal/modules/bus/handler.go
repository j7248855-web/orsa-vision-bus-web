package bus

import (
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Логика добавления нового автобуса и его девайсов
func RegisterBus(ctx *gin.Context, conn *sqlx.DB) {
	var bus models.Bus
	if err := ctx.ShouldBindJSON(&bus); err != nil {
		ctx.JSON(400, gin.H{"error": "Непредвиденная ошибка:", "details": err.Error()})
		return
	}
	var exists bool

	//Проверка на уникальность, есть ли в базе уже подобный автобус
	queryUniqueBus := `SELECT EXISTS(SELECT 1 FROM buses WHERE city=$1 AND bus_number=$2)`
	err := conn.QueryRowContext(ctx, queryUniqueBus, bus.City, bus.BusNumber).Scan(&exists)

	if err != nil {
		log.Println("Ошибка при проверке уникальности:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка базы данных"})
		return
	}

	if exists {
		log.Printf("Попытка дубликата: автобус %s уже есть в городе %s", bus.BusNumber, bus.City)
		ctx.JSON(400, gin.H{"error": "Данный автобус уже существует в указанном городе"})
		return
	}
	tx, err := conn.Beginx()
	if err != nil {
		log.Println("Ошибка в транзакции", err)
		ctx.JSON(500, gin.H{"error": "Ошибка транзакции при создании нового автобуса"})
		return
	}
	defer tx.Rollback()
	var lastBusID int

	//Добавление в базу данных нового автобуса
	queryBus := `
        INSERT INTO buses (bus_number, route_number, status, city) 
        VALUES ($1, $2, $3, $4) 
        RETURNING id`

	err = tx.QueryRow(queryBus, bus.BusNumber, bus.RouteNumber, bus.Status, bus.City).Scan(&lastBusID)
	if err != nil {
		log.Println("Ошибка добавления автобуса:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка сохранения автобуса: " + err.Error()})
		return
	}

	if len(bus.Devices) > 0 {
		for _, dev := range bus.Devices {
			queryDev := `
                INSERT INTO devices (device_ip, type, status, bus_id) 
                VALUES ($1, $2, $3, $4)`

			_, err = tx.ExecContext(ctx, queryDev, dev.DeviceIP, dev.Type, dev.Status, lastBusID)
			if err != nil {
				log.Println("Ошибка добавления девайса:", err)
				ctx.JSON(500, gin.H{"error": "Ошибка сохранения девайса: " + err.Error()})
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		ctx.JSON(500, gin.H{"error": "Не удалось подтвердить транзакцию"})
		return
	}

	ctx.JSON(200, gin.H{"status": "success", "id": lastBusID})
}

// Отправка всех автобусов из базы данных
func GetBuses(ctx *gin.Context, conn *sqlx.DB) {
	var buses []models.Bus
	var devices []models.Device
	busMap := make(map[string]*models.Bus)
	//Достаём автобусы
	err := conn.Select(&buses, "SELECT id, bus_number, route_number, status FROM buses")
	if err != nil {
		log.Println("Ошибка получения автобусов:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка базы данных при поиске автобусов"})
		return
	}

	//Достаём данные девайсов
	err = conn.Select(&devices, "SELECT id, device_ip, type, status")
	if err != nil {
		log.Println("Ошибка получения ltdfqcjd:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка базы данных при поиске девайсов", "details": err.Error()})
		return
	}
	//Сопоставление данных от автобуса, с данными от девайсов

	for value := range buses {
		busMap[buses[value].ID] = &buses[value]
	}
	for _, value := range devices {
		if bus, ok := busMap[value.BusID]; ok {
			bus.Devices = append(bus.Devices, value)
		}
	}

	ctx.JSON(200, buses)
}

// Редактирование данных автобуса
func EditBus(ctx *gin.Context, conn *sqlx.DB) {
	var updatedBus models.Bus

	if err := ctx.ShouldBindJSON(&updatedBus); err != nil {
		ctx.JSON(400, gin.H{"error": "Непредвиденная ошибка:", "details": err.Error()})
		return
	}

	query := `
		UPDATE buses 
		SET bus_number = $1, route_number = $2, status = $3, city = $4
		WHERE id = $5`

	result, err := conn.ExecContext(ctx, query,
		updatedBus.BusNumber,
		updatedBus.RouteNumber,
		updatedBus.Status,
		updatedBus.City,
		updatedBus.ID,
	)

	if err != nil {
		ctx.JSON(500, gin.H{"error": "Проблема обновления данных: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		ctx.JSON(404, gin.H{"error": "Автобус с таким ID не найден"})
		return
	}

	ctx.JSON(200, gin.H{"message": "Данные автобуса обновлены"})
}

// Удаление автобуса
func RemoveBus(ctx *gin.Context, conn *sqlx.DB) {
	busID := ctx.Param("bus_id")

	query := `DELETE FROM buses WHERE id = $1`
	result, err := conn.ExecContext(ctx, query, busID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Ошибка при удалении автобуса: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		ctx.JSON(404, gin.H{"error": "Автобус с таким ID не найден"})
		return
	}

	ctx.JSON(200, gin.H{"message": "Автобус успешно удален"})
}
