package serviceb

import (
	"bytes"
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

type ViaCEPClient struct {
	httpClient *http.Client
	baseURL    string
	tracer     trace.Tracer
}

func NewViaCEPClient(baseURL string, tracer trace.Tracer) *ViaCEPClient {
	if baseURL == "" {
		baseURL = "https://viacep.com.br"
	}
	return &ViaCEPClient{
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		baseURL: strings.TrimRight(baseURL, "/"),
		tracer:  tracer,
	}
}

type viaCEPResponse struct {
	Localidade string       `json:"localidade"`
	UF         string       `json:"uf"`
	Erro       flexibleBool `json:"erro"`
}

type flexibleBool bool

func (b *flexibleBool) UnmarshalJSON(data []byte) error {
	trimmed := string(bytes.Trim(data, ` "`))
	if trimmed == "true" {
		*b = true
		return nil
	}
	if trimmed == "false" || trimmed == "" {
		*b = false
		return nil
	}
	return nil
}

func (c *ViaCEPClient) GetCity(ctx context.Context, zipcode string) (string, error) {
	ctx, span := c.tracer.Start(ctx, "busca-cep")
	defer span.End()

	endpoint, err := url.JoinPath(c.baseURL, "ws", zipcode, "json")
	if err != nil {
		return "", fmt.Errorf("url invalida para viacep: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisicao viacep: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("falha de comunicacao com viacep: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", entity.ErrZipcodeNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status inesperado do viacep: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("falha ao ler resposta do viacep: %w", err)
	}

	var res viaCEPResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("falha ao decodificar json do viacep: %w", err)
	}

	if res.Erro || strings.TrimSpace(res.Localidade) == "" {
		return "", entity.ErrZipcodeNotFound
	}

	return strings.TrimSpace(res.Localidade), nil
}
