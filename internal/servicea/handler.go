package servicea

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"clima-cep-otel/internal/entity"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	serviceBURL string
	httpClient  *http.Client
	tracer      trace.Tracer
}

func NewHandler(serviceBURL string, tracer trace.Tracer) *Handler {
	if serviceBURL == "" {
		serviceBURL = "http://service-b:8081"
	}
	return &Handler{
		serviceBURL: strings.TrimRight(serviceBURL, "/"),
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		tracer: tracer,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", h.handlePost)
	mux.HandleFunc("POST /cep", h.handlePost)
	mux.HandleFunc("POST /clima", h.handlePost)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	return mux
}

func (h *Handler) handlePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := h.tracer.Start(ctx, "service-a-receive-request")
	defer span.End()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusUnprocessableEntity, "invalid zipcode")
		return
	}

	var req entity.CEPRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		h.writeError(w, http.StatusUnprocessableEntity, "invalid zipcode")
		return
	}

	if err := entity.ValidateZipcode(req.CEP); err != nil {
		h.writeError(w, http.StatusUnprocessableEntity, "invalid zipcode")
		return
	}

	serviceBEndpoint := fmt.Sprintf("%s/", h.serviceBURL)
	payload, _ := json.Marshal(req)

	reqB, err := http.NewRequestWithContext(ctx, http.MethodPost, serviceBEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "erro ao criar requisicao para o servico B")
		return
	}
	reqB.Header.Set("Content-Type", "application/json")

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(reqB.Header))

	respB, err := h.httpClient.Do(reqB)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, fmt.Sprintf("erro de comunicacao com servico B: %v", err))
		return
	}
	defer respB.Body.Close()

	bodyB, err := io.ReadAll(respB.Body)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "erro ao ler resposta do servico B")
		return
	}

	if ct := respB.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(respB.StatusCode)
	_, _ = w.Write(bodyB)
}

func (h *Handler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(message))
}
