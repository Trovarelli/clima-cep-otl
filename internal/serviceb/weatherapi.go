package serviceb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"clima-cep-otel/internal/entity"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

type WeatherAPIClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	tracer     trace.Tracer
}

func NewWeatherAPIClient(baseURL, apiKey string, tracer trace.Tracer) *WeatherAPIClient {
	if baseURL == "" {
		baseURL = "https://api.weatherapi.com/v1"
	}
	return &WeatherAPIClient{
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  strings.TrimSpace(apiKey),
		tracer:  tracer,
	}
}

type weatherAPIResponse struct {
	Current struct {
		TempC *float64 `json:"temp_c"`
	} `json:"current"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *WeatherAPIClient) GetTemperatureCelsius(ctx context.Context, city string) (float64, error) {
	ctx, span := c.tracer.Start(ctx, "busca-temperatura")
	defer span.End()

	if c.apiKey == "" {
		return 0, fmt.Errorf("chave da WeatherAPI nao configurada")
	}

	endpoint, err := url.JoinPath(c.baseURL, "current.json")
	if err != nil {
		return 0, fmt.Errorf("construcao de url invalida: %w", err)
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return 0, fmt.Errorf("parse de url invalida: %w", err)
	}

	q := u.Query()
	q.Set("key", c.apiKey)
	q.Set("q", city)
	q.Set("aqi", "no")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("criacao da requisicao weatherapi: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("comunicacao com weatherapi falhou: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("falha ao ler resposta da weatherapi: %w", err)
	}

	var res weatherAPIResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return 0, fmt.Errorf("falha ao decodificar json da weatherapi: %w", err)
	}

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound {
		if res.Error != nil && res.Error.Code == 1006 {
			return 0, entity.ErrZipcodeNotFound
		}
		return 0, fmt.Errorf("erro da weatherapi status %d: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status inesperado da weatherapi: %d", resp.StatusCode)
	}

	if res.Current.TempC == nil {
		return 0, fmt.Errorf("temperatura nao encontrada no corpo de resposta da weatherapi")
	}

	return *res.Current.TempC, nil
}
