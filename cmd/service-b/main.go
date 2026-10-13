package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"clima-cep-otel/internal/serviceb"
	"clima-cep-otel/internal/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	weatherAPIKey := os.Getenv("WEATHER_API_KEY")
	if weatherAPIKey == "" {
		logger.Warn("ATENCAO: WEATHER_API_KEY nao configurada no Servico B!")
	}

	otelCollectorAddr := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelCollectorAddr == "" {
		otelCollectorAddr = os.Getenv("OTEL_COLLECTOR_ADDR")
	}
	if otelCollectorAddr == "" {
		otelCollectorAddr = "otel-collector:4317"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracer, shutdownTracer, err := telemetry.InitTracer(ctx, "service-b", otelCollectorAddr)
	if err != nil {
		logger.Error("falha ao inicializar tracer no Servico B", "erro", err)
	} else {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := shutdownTracer(shutdownCtx); err != nil {
				logger.Error("erro ao finalizar tracer", "erro", err)
			}
		}()
	}

	viaCEPClient := serviceb.NewViaCEPClient("", tracer)
	weatherClient := serviceb.NewWeatherAPIClient("", weatherAPIKey, tracer)
	handler := serviceb.NewHandler(viaCEPClient, weatherClient, tracer)

	otelHandler := otelhttp.NewHandler(handler.Routes(), "service-b-http")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           otelHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Serviço B iniciado", "porta", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("falha ao iniciar servidor HTTP do Serviço B", "erro", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("desligando Serviço B graciosamente...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("erro ao desligar servidor", "erro", err)
	}
	logger.Info("Serviço B finalizado com sucesso.")
}
