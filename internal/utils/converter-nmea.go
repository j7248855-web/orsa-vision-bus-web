package utils

import (
	"math"
	"strconv"
)

func ConvertNMEAToDecimal(nmea string, direction string) float64 {
	val, _ := strconv.ParseFloat(nmea, 64)
	if val == 0 {
		return 0
	}

	// Делим на 100, чтобы отделить градусы (целая часть) от минут
	// Например: 5020.89669 / 100 = 50.2089669
	degrees := math.Floor(val / 100)
	// Минуты: 5020.89669 - (50 * 100) = 20.89669
	minutes := val - (degrees * 100)

	// Итого: Градусы + (Минуты / 60)
	decimal := degrees + (minutes / 60.0)

	// Если направление S (юг) или W (запад), делаем число отрицательным
	if direction == "S" || direction == "W" {
		decimal = -decimal
	}

	return decimal
}
