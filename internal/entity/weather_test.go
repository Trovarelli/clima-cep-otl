package entity

import (
	"testing"
)

func TestValidateZipcode(t *testing.T) {
	tests := []struct {
		name    string
		zipcode string
		wantErr bool
	}{
		{"Valid CEP", "01001000", false},
		{"Valid CEP 8 digits", "29902555", false},
		{"Too short", "1234567", true},
		{"Too long", "123456789", true},
		{"Contains letters", "0100100a", true},
		{"Contains hyphen", "01001-000", true},
		{"Empty", "", true},
		{"Spaces", "0100 000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateZipcode(tt.zipcode)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateZipcode(%q) error = %v, wantErr %v", tt.zipcode, err, tt.wantErr)
			}
		})
	}
}

func TestTemperatureConversions(t *testing.T) {
	celsius := 28.5
	expectedF := 83.3
	expectedK := 301.65

	gotF := CelsiusToFahrenheit(celsius)
	if gotF != expectedF {
		t.Errorf("CelsiusToFahrenheit(%v) = %v; want %v", celsius, gotF, expectedF)
	}

	gotK := CelsiusToKelvin(celsius)
	if gotK != expectedK {
		t.Errorf("CelsiusToKelvin(%v) = %v; want %v", celsius, gotK, expectedK)
	}
}
