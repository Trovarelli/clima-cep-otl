package serviceb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"clima-cep-otel/internal/entity"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type LocationProvider interface {
	GetCity(ctx context.Context, zipcode string) (string, error)
}

type WeatherProvider interface {
	GetTemperatureCelsius(ctx context.Context, city string) (float64, error)
}

type Handler struct {
	locationProvider LocationProvider
	weatherProvider  WeatherProvider
	tracer           trace.Tracer
}

func NewHandler(loc LocationProvider, weather WeatherProvider, tracer trace.Tracer) *Handler {
	return &Handler{
		locationProvider: loc,
		weatherProvider:  weather,
		tracer:           tracer,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", h.handleRequest)
	mux.HandleFunc("POST /cep", h.handleRequest)
	mux.HandleFunc("POST /clima", h.handleRequest)
	mux.HandleFunc("GET /{cep}", h.handleRequest)
	mux.HandleFunc("GET /clima/{cep}", h.handleRequest)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	return mux
}

func (h *Handler) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Extrai contexto propagado pelo Serviço A (W3C traceparent)
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	ctx, span := h.tracer.Start(ctx, "service-b-orchestration")
	defer span.End()

	cep := h.extractZipcode(r)
	if err := entity.ValidateZipcode(cep); err != nil {
		h.writeError(w, http.StatusUnprocessableEntity, "invalid zipcode")
		return
	}

	// 1. Consulta cidade no ViaCEP (com span manual interno busca-cep)
	city, err := h.locationProvider.GetCity(ctx, cep)
	if err != nil {
		if errors.Is(err, entity.ErrZipcodeNotFound) {
			h.writeError(w, http.StatusNotFound, "can not find zipcode")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "erro ao consultar localizacao")
		return
	}

	// 2. Consulta temperatura na WeatherAPI (com span manual interno busca-temperatura)
	tempC, err := h.weatherProvider.GetTemperatureCelsius(ctx, city)
	if err != nil {
		if errors.Is(err, entity.ErrZipcodeNotFound) {
			h.writeError(w, http.StatusNotFound, "can not find zipcode")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "erro ao consultar clima")
		return
	}

	resp := entity.WeatherResponse{
		City:  city,
		TempC: tempC,
		TempF: entity.CelsiusToFahrenheit(tempC),
		TempK: entity.CelsiusToKelvin(tempC),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) extractZipcode(r *http.Request) string {
	if r.Method == http.MethodPost && r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			var req entity.CEPRequest
			if err := json.Unmarshal(bodyBytes, &req); err == nil && req.CEP != "" {
				return strings.TrimSpace(req.CEP)
			}
		}
	}

	if cep := strings.TrimSpace(r.PathValue("cep")); cep != "" {
		return cep
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if last != "" && last != "clima" && last != "cep" && last != "health" {
			return last
		}
	}

	return ""
}

func (h *Handler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(message))
}
