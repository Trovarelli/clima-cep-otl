package serviceb

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"clima-cep-otel/internal/entity"

	"go.opentelemetry.io/otel/trace/noop"
)

type mockLocationProvider struct {
	city string
	err  error
}

func (m *mockLocationProvider) GetCity(ctx context.Context, zipcode string) (string, error) {
	return m.city, m.err
}

type mockWeatherProvider struct {
	tempC float64
	err   error
}

func (m *mockWeatherProvider) GetTemperatureCelsius(ctx context.Context, city string) (float64, error) {
	return m.tempC, m.err
}

func TestServiceBHandler_Success(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	loc := &mockLocationProvider{city: "São Paulo"}
	weather := &mockWeatherProvider{tempC: 28.5}

	handler := NewHandler(loc, weather, tracer)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"cep":"01001000"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res entity.WeatherResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if res.City != "São Paulo" {
		t.Errorf("expected city 'São Paulo', got %q", res.City)
	}
	if res.TempC != 28.5 {
		t.Errorf("expected temp_C 28.5, got %v", res.TempC)
	}
	if res.TempF != 83.3 {
		t.Errorf("expected temp_F 83.3, got %v", res.TempF)
	}
	if res.TempK != 301.65 {
		t.Errorf("expected temp_K 301.65, got %v", res.TempK)
	}
}

func TestServiceBHandler_InvalidZipcode(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	loc := &mockLocationProvider{city: "São Paulo"}
	weather := &mockWeatherProvider{tempC: 28.5}

	handler := NewHandler(loc, weather, tracer)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"cep":"12345"}`))
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", rec.Code)
	}
	if rec.Body.String() != "invalid zipcode" {
		t.Errorf("expected 'invalid zipcode', got %q", rec.Body.String())
	}
}

func TestServiceBHandler_NotFound(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	loc := &mockLocationProvider{err: entity.ErrZipcodeNotFound}
	weather := &mockWeatherProvider{}

	handler := NewHandler(loc, weather, tracer)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"cep":"99999999"}`))
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
	if rec.Body.String() != "can not find zipcode" {
		t.Errorf("expected 'can not find zipcode', got %q", rec.Body.String())
	}
}
