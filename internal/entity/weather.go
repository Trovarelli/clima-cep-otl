package entity

import (
	"errors"
	"math"
	"unicode"
)

var (
	ErrInvalidZipcode  = errors.New("invalid zipcode")
	ErrZipcodeNotFound = errors.New("can not find zipcode")
)

type CEPRequest struct {
	CEP string `json:"cep"`
}

type WeatherResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func ValidateZipcode(zipcode string) error {
	if len(zipcode) != 8 {
		return ErrInvalidZipcode
	}
	for _, r := range zipcode {
		if !unicode.IsDigit(r) {
			return ErrInvalidZipcode
		}
	}
	return nil
}

func CelsiusToFahrenheit(c float64) float64 {
	return math.Round((c*1.8+32)*100) / 100
}

func CelsiusToKelvin(c float64) float64 {
	return math.Round((c+273.15)*100) / 100
}
