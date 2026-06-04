package bus

import (
	"fmt"
	"log"
	"orsavisionweb/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Логика добавления нового автобуса и его девайсов
func RegisterBus(ctx *gin.Context, conn *sqlx.DB) {
	var bus models.Bus
	if err := ctx.ShouldBindJSON(&bus); err != nil {
		log.Println("Не получилось распарсить данные:", err)
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

	err = tx.QueryRowContext(ctx, `
        INSERT INTO buses (bus_number, route_number, status, city, sequence_number) 
        VALUES ($1, $2, $3, $4, $5) 
        RETURNING id`, bus.BusNumber, bus.RouteNumber, bus.Status, bus.City, bus.SequenceNumber).Scan(&lastBusID)
	if err != nil {
		log.Println("Ошибка добавления автобуса:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка сохранения автобуса: " + err.Error()})
		return
	}
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
	var schedules []models.ScheduleBus

	busMap := make(map[string]*models.Bus)
	seqMap := make(map[int]*models.Bus)

	// Достаём автобусы
	err := conn.SelectContext(ctx, &buses, "SELECT id, bus_number, route_number, status, city, sequence_number FROM buses")
	if err != nil {
		log.Println("Ошибка получения автобусов:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка базы данных при поиске автобусов"})
		return
	}

	// Достаём данные девайсов
	err = conn.SelectContext(ctx, &devices, "SELECT id, bus_id, device_ip, type, status FROM devices")
	if err != nil {
		log.Println("Ошибка получения девайсов:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка базы данных при поиске девайсов", "details": err.Error()})
		return
	}

	scheduleQuery := `
        SELECT DISTINCT ON (sequence_number) sequence_number, arrival_time, departure_time 
        FROM schedules 
        ORDER BY sequence_number, arrival_time ASC
    `
	err = conn.SelectContext(ctx, &schedules, scheduleQuery)
	if err != nil {
		log.Println("Ошибка получения расписания:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка базы данных при поиске расписания", "details": err.Error()})
		return
	}

	// Заполняем мапы для быстрого поиска
	for value := range buses {
		busMap[buses[value].ID] = &buses[value]
		seqMap[buses[value].SequenceNumber] = &buses[value]
	}

	// Сопоставление девайсов по bus_id
	for _, value := range devices {
		if bus, ok := busMap[value.BusID]; ok {
			bus.Devices = append(bus.Devices, value)
		}
	}

	// Сопоставление расписания по sequence_number
	for _, value := range schedules {
		if bus, ok := seqMap[value.SequenceNumber]; ok {
			bus.Schedule = append(bus.Schedule, value)
		}
	}

	ctx.JSON(200, buses)
}

// Редактирование данных автобуса
func EditBus(ctx *gin.Context, conn *sqlx.DB) {
	var updatedBus models.Bus

	if err := ctx.ShouldBindJSON(&updatedBus); err != nil {
		log.Println("Не получилось распарсить данные:", err)
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
	if len(updatedBus.Devices) > 0 {

		for _, device := range updatedBus.Devices {
			device.BusID = updatedBus.ID
			_, err := conn.ExecContext(ctx, "DELETE FROM devices WHERE bus_id = $1", updatedBus.ID)
			if err != nil {
				log.Println("Не удалось удалить старые устройства:", err)
				ctx.JSON(500, gin.H{"error": "Проблема очистки старых устройств: " + err.Error()})
				return
			}

			insertQuery := `
            INSERT INTO devices (bus_id, device_ip, type, status) 
            VALUES (:bus_id, :device_ip, :type, :status)`

			for _, device := range updatedBus.Devices {
				device.BusID = updatedBus.ID // Привязываем к нашему автобусу

				_, err := conn.NamedExecContext(ctx, insertQuery, device)
				if err != nil {
					log.Println("Не удалось добавить обновленное устройство:", err)
					ctx.JSON(500, gin.H{"error": "Проблема сохранения устройств: " + err.Error()})
					return
				}
			}
		}
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		ctx.JSON(404, gin.H{"error": "Автобус с таким ID не найден"})
		return
	}

	ctx.JSON(200, gin.H{"status": "success"})
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

// Данные о графике
func DataBus(ctx *gin.Context, conn *sqlx.DB) {
	routeNumber := ctx.Param("route_number")

	var routeID int
	err := conn.GetContext(ctx, &routeID, "SELECT id FROM routes WHERE route_number = $1 LIMIT 1", routeNumber)
	if err != nil {
		log.Println("Маршрут не найден или ошибка БД:", err)
		ctx.JSON(404, gin.H{"error": "Маршрут с таким номером не найден"})
		return
	}

	var schedules []models.TripSteps
	err = conn.SelectContext(ctx, &schedules, `
		SELECT DISTINCT ON (sequence_number) route_id, sequence_number, departure_time, arrival_time 
		FROM schedules 
		WHERE route_id = $1
		ORDER BY sequence_number ASC, departure_time ASC`, routeID)
	if err != nil {
		log.Println("Не получилось распарсить данные расписания:", err)
		ctx.JSON(500, gin.H{"error": "Ошибка при получении расписания: " + err.Error()})
		return
	}
	fmt.Println("Данные из базы", schedules)
	ctx.JSON(200, gin.H{
		"schedules": schedules,
	})
}

// Тревожная кнопка
func EmergencyButton(ctx *gin.Context, conn *sqlx.DB) {
	var emergencyAlert models.EmergencyInformation
	err := conn.SelectContext(ctx, &emergencyAlert, "SELECT bus_id, bus_number, route_number, emergency_at FROM buses_emergencies")
	if err != nil {
		log.Println("Не удалось достать из базы данные:", err)
		ctx.JSON(500, gin.H{"status": "error", "details": err.Error()})
		return
	}
	ctx.JSON(200, emergencyAlert)
}
